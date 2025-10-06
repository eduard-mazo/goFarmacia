package backend

import (
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecordInfo struct {
	Key       string
	UpdatedAt time.Time
	Record    interface{}
}

func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

func (d *Db) SincronizacionInteligente() {
	if !d.isRemoteDBAvailable() {
		d.Log.Warnf("Modo offline: la base de datos remota no está disponible, se omite la sincronización.")
		return
	}
	d.Log.Infof("[INICIO]: Sincronización Inteligente")

	d.sincronizarModelo(&[]Vendedor{}, "Vendedores")
	d.sincronizarModelo(&[]Cliente{}, "Clientes")
	d.sincronizarModelo(&[]Proveedor{}, "Proveedores")
	d.sincronizarModelo(&[]Producto{}, "Productos")

	// --- ORDEN DE SINCRONIZACIÓN CORREGIDO ---
	// 1. Primero, enviamos nuestras operaciones locales al servidor.
	d.SincronizarOperacionesStockHaciaRemoto()

	// 2. Segundo, descargamos las operaciones de otros clientes que no tengamos.
	if err := d.sincronizarOperacionesHaciaLocal(); err == nil {
		// 3. Si se descargaron operaciones, es crucial recalcular el stock local.
		d.Log.Info("Recalculando stock de todos los productos locales tras la sincronización.")
		var productosLocales []Producto
		d.LocalDB.Find(&productosLocales)
		tx := d.LocalDB.Begin()
		if tx.Error != nil {
			d.Log.Errorf("Error al iniciar la transacción para el recálculo de stock: %v", tx.Error)
			return
		}
		defer tx.Rollback()

		for _, p := range productosLocales {
			if err := RecalcularYActualizarStock(tx, p.ID); err != nil {
				d.Log.Errorf("Error al recalcular el stock para el producto ID %d: %v", p.ID, err)
			}
		}

		if err := tx.Commit().Error; err != nil {
			d.Log.Errorf("Error al confirmar la transacción de recálculo de stock: %v", err)
		}
	}
	d.sincronizarTransaccionesHaciaLocal()

	d.Log.Infof("[FIN]: Sincronización Inteligente")
}

func (d *Db) sincronizarOperacionesHaciaLocal() error {
	if !d.isRemoteDBAvailable() {
		return nil // No hacer nada si estamos offline
	}
	d.Log.Info("[SINCRONIZANDO]: Operaciones de Stock desde Remoto -> Local")

	var localUUIDs []string
	if err := d.LocalDB.Model(&OperacionStock{}).Pluck("uuid", &localUUIDs).Error; err != nil {
		d.Log.Errorf("Error obteniendo UUIDs locales: %v", err)
		return err
	}

	var nuevasOperaciones []OperacionStock
	query := d.RemoteDB.Model(&OperacionStock{})
	if len(localUUIDs) > 0 {
		query = query.Where("uuid NOT IN ?", localUUIDs)
	}
	if err := query.Find(&nuevasOperaciones).Error; err != nil {
		d.Log.Errorf("Error buscando nuevas operaciones en remoto: %v", err)
		return err
	}

	if len(nuevasOperaciones) > 0 {
		d.Log.Infof("Descargando %d nueva(s) operaciones de stock...", len(nuevasOperaciones))
		if err := d.LocalDB.Create(&nuevasOperaciones).Error; err != nil {
			d.Log.Errorf("Error insertando nuevas operaciones en local: %v", err)
			return err
		}
	} else {
		d.Log.Info("No se encontraron nuevas operaciones de stock en el servidor.")
	}
	return nil
}

func (d *Db) sincronizarModelo(modeloSlice interface{}, nombreModelo string) {
	d.Log.Infof("[SINCRONIZANDO]:\t%s\n", nombreModelo)

	// --- INICIO DE LA MODIFICACIÓN ---
	// Usamos Unscoped() para incluir registros con borrado lógico en la comparación
	if err := d.LocalDB.Unscoped().Find(modeloSlice).Error; err != nil {
		d.Log.Errorf("Cargando %s desde LOCAL: %v", nombreModelo, err)
		return
	}
	// --- FIN DE LA MODIFICACIÓN ---
	localRecords := sliceToInterfaceSlice(modeloSlice)
	if localRecords == nil {
		d.Log.Warnf("Sin registros en LOCAL para %s", nombreModelo)
	}

	// --- INICIO DE LA MODIFICACIÓN ---
	// Usamos Unscoped() también para la base de datos remota
	if err := d.RemoteDB.Unscoped().Find(modeloSlice).Error; err != nil {
		d.Log.Errorf("Cargando %s desde REMOTO: %v", nombreModelo, err)
		return
	}
	// --- FIN DE LA MODIFICACIÓN ---
	remoteRecords := sliceToInterfaceSlice(modeloSlice)
	if remoteRecords == nil {
		d.Log.Infof("Warning: no remoteRecords parsed for %s", nombreModelo)
	}

	localMap := make(map[string]RecordInfo)
	for _, rec := range localRecords {
		info := getRecordInfo(rec)
		if info.Key != "" {
			localMap[info.Key] = info
		}
	}

	remoteMap := make(map[string]RecordInfo)
	for _, rec := range remoteRecords {
		info := getRecordInfo(rec)
		if info.Key != "" {
			remoteMap[info.Key] = info
		}
	}

	var paraCrearEnRemoto, paraActualizarEnRemoto, paraActualizarEnLocal []interface{}
	allKeys := getCombinedKeys(localMap, remoteMap)

	for key := range allKeys {
		local, localExists := localMap[key]
		remote, remoteExists := remoteMap[key]

		if localExists && remoteExists {
			localTime := local.UpdatedAt.Truncate(time.Millisecond)
			remoteTime := remote.UpdatedAt.Truncate(time.Millisecond)

			if localTime.After(remoteTime) {
				paraActualizarEnRemoto = append(paraActualizarEnRemoto, local.Record)
			} else if remoteTime.After(localTime) {
				paraActualizarEnLocal = append(paraActualizarEnLocal, remote.Record)
			}
		} else if localExists && !remoteExists {
			paraCrearEnRemoto = append(paraCrearEnRemoto, local.Record)
		} else if !localExists && remoteExists {
			paraActualizarEnLocal = append(paraActualizarEnLocal, remote.Record)
		}
	}

	if len(paraCrearEnRemoto) > 0 {
		d.Log.Infof("[%s]\tCreando %d nuevo(s) registro(s) en REMOTO...", nombreModelo, len(paraCrearEnRemoto))
		d.crearLoteEnRemoto(d.RemoteDB, nombreModelo, paraCrearEnRemoto)
	}

	if len(paraActualizarEnRemoto) > 0 {
		d.Log.Infof("[%s]\tActualizando %d registro(s) en REMOTO...", nombreModelo, len(paraActualizarEnRemoto))
		d.actualizarLoteEnRemoto(d.RemoteDB, nombreModelo, paraActualizarEnRemoto)
	}

	if len(paraActualizarEnLocal) > 0 {
		d.Log.Infof("[%s]\tSincronizando %d registro(s) a LOCAL...", nombreModelo, len(paraActualizarEnLocal))
		d.ejecutarLote(d.LocalDB, nombreModelo, paraActualizarEnLocal)
	}

	if len(paraActualizarEnLocal) == 0 && len(paraCrearEnRemoto) == 0 && len(paraActualizarEnRemoto) == 0 {
		d.Log.Infof("[%s]\tNo se encuentran diferencias... MODELO sincronizado", nombreModelo)
	}
}

func (d *Db) ejecutarLote(db *gorm.DB, nombreModelo string, lote []interface{}) {
	if len(lote) == 0 {
		return
	}

	var claves []string
	for _, rec := range lote {
		info := getRecordInfo(rec)
		if info.Key != "" {
			claves = append(claves, info.Key)
		}
	}
	if len(claves) > 0 {
		log.Printf("[%s] Claves a sincronizar: %s", nombreModelo, strings.Join(claves, ", "))
	}

	tx := db.Begin()
	if tx.Error != nil {
		log.Printf("Error starting transaction for %s: %v", nombreModelo, tx.Error)
		return
	}

	for _, record := range lote {
		conflictColumn := getConflictColumn(record)
		if conflictColumn != "" {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: conflictColumn}},
				UpdateAll: true,
			}).Create(record).Error; err != nil {
				log.Printf("Error saving record for %s (conflict=%s): %v", nombreModelo, conflictColumn, err)
				continue
			}
		} else {
			if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(record).Error; err != nil {
				log.Printf("Error saving record for %s: %v", nombreModelo, err)
				continue
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction for %s: %v", nombreModelo, err)
	}
}

func (d *Db) crearLoteEnRemoto(db *gorm.DB, nombreModelo string, lote []interface{}) {
	if len(lote) == 0 {
		return
	}

	for _, record := range lote {
		tx := db.Begin()
		if tx.Error != nil {
			d.Log.Errorf("Error iniciando transacción para un registro de %s: %v", nombreModelo, tx.Error)
			continue
		}

		conflictColumn := getConflictColumn(record)
		if conflictColumn == "" {
			d.Log.Warnf("No se encontró columna de conflicto para el modelo %s, se omitirá la creación.", nombreModelo)
			tx.Rollback()
			continue
		}

		switch r := record.(type) {
		case *Producto:
			r.ID = 0
		case *Vendedor:
			r.ID = 0
		case *Cliente:
			r.ID = 0
		case *Proveedor:
			r.ID = 0
		}

		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: conflictColumn}},
			DoNothing: true,
		}).Create(record).Error

		if err != nil {
			d.Log.Errorf("Error en la creación de registro para %s en REMOTO: %v", nombreModelo, err)
			tx.Rollback()
			continue
		}

		if err := tx.Commit().Error; err != nil {
			d.Log.Errorf("Error al confirmar transacción para un registro de %s: %v", nombreModelo, err)
		}
	}
}

func (d *Db) actualizarLoteEnRemoto(db *gorm.DB, nombreModelo string, lote []interface{}) {
	tx := db.Begin()
	if tx.Error != nil {
		d.Log.Errorf("Error iniciando transacción de actualización para %s en REMOTO: %v", nombreModelo, tx.Error)
		return
	}
	defer tx.Rollback()

	for _, record := range lote {
		switch r := record.(type) {
		case *Producto:
			result := tx.Model(&Producto{}).Where("codigo = ?", r.Codigo).Updates(map[string]interface{}{
				"nombre":       r.Nombre,
				"precio_venta": r.PrecioVenta,
				"updated_at":   time.Now(),
				"deleted_at":   r.DeletedAt,
			})
			if result.Error != nil {
				d.Log.Errorf("Error actualizando producto remoto con código %s: %v", r.Codigo, result.Error)
			}
		case *Vendedor:
			result := tx.Model(&Vendedor{}).Where("cedula = ?", r.Cedula).Updates(r)
			if result.Error != nil {
				d.Log.Errorf("Error actualizando vendedor remoto con cédula %s: %v", r.Cedula, result.Error)
			}
		case *Cliente:
			result := tx.Model(&Cliente{}).Where("numero_id = ?", r.NumeroID).Updates(r)
			if result.Error != nil {
				d.Log.Errorf("Error actualizando cliente remoto con NumeroID %s: %v", r.NumeroID, result.Error)
			}
		case *Proveedor:
			result := tx.Model(&Proveedor{}).Where("nombre = ?", r.Nombre).Updates(r)
			if result.Error != nil {
				d.Log.Errorf("Error actualizando proveedor remoto con nombre %s: %v", r.Nombre, result.Error)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		d.Log.Errorf("Error al confirmar transacción de actualización para %s en REMOTO: %v", nombreModelo, err)
	}
}

func (d *Db) SincronizarOperacionesStockHaciaRemoto() {
	if !d.isRemoteDBAvailable() {
		return
	}
	d.Log.Info("[SINCRONIZANDO]: Operaciones de Stock desde Local -> Remoto")

	var ops []OperacionStock
	if err := d.LocalDB.Where("sincronizado = ?", false).Find(&ops).Error; err != nil {
		log.Printf("Error cargando operaciones de stock pendientes: %v", err)
		return
	}
	if len(ops) == 0 {
		return
	}

	log.Printf("Enviando %d operación(es) de stock al servidor remoto...", len(ops))

	uniqueMap := make(map[string]OperacionStock)
	for _, op := range ops {
		op.Sincronizado = false
		existing, ok := uniqueMap[op.UUID]
		if !ok || op.Timestamp.After(existing.Timestamp) {
			uniqueMap[op.UUID] = op
		}
	}
	finalOps := make([]OperacionStock, 0, len(uniqueMap))
	for _, op := range uniqueMap {
		op.ID = 0
		finalOps = append(finalOps, op)
	}

	tx := d.RemoteDB.Begin()
	if tx.Error != nil {
		log.Printf("Error iniciando transacción remota para operaciones de stock: %v", tx.Error)
		return
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoNothing: true, // Usamos DoNothing para no sobreescribir si ya existe
	}).Create(&finalOps).Error; err != nil {
		log.Printf("Error en batch upsert de operaciones de stock: %v", err)
		tx.Rollback()
		return
	}

	productoIDs := make(map[uint]bool)
	for _, op := range finalOps {
		productoIDs[op.ProductoID] = true
	}

	// ¡CRÍTICO! Recalcular el stock en el remoto usando los datos DEL REMOTO.
	for id := range productoIDs {
		var stockRemoto int
		tx.Model(&OperacionStock{}).Where("producto_id = ?", id).Select("COALESCE(SUM(cantidad_cambio), 0)").Row().Scan(&stockRemoto)
		if err := tx.Model(&Producto{}).Where("id = ?", id).Update("stock", stockRemoto).Error; err != nil {
			log.Printf("Error actualizando stock remoto para producto %d: %v", id, err)
			tx.Rollback()
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error confirmando transacción remota para operaciones de stock: %v", err)
		return
	}

	var uuids []string
	for _, op := range finalOps {
		uuids = append(uuids, op.UUID)
	}
	if err := d.LocalDB.Model(&OperacionStock{}).
		Where("uuid IN ?", uuids).
		Update("sincronizado", true).Error; err != nil {
		log.Printf("Error marcando operaciones de stock como sincronizadas en local: %v", err)
	}
}

func (d *Db) sincronizarTransaccionesHaciaLocal() {
	var ultimaFacturaLocal Factura
	d.LocalDB.Order("created_at desc").Limit(1).Find(&ultimaFacturaLocal)

	var facturasNuevas []Factura
	queryFacturas := d.RemoteDB.Preload("Detalles")
	if !ultimaFacturaLocal.CreatedAt.IsZero() {
		queryFacturas = queryFacturas.Where("created_at > ?", ultimaFacturaLocal.CreatedAt)
	}
	if err := queryFacturas.Find(&facturasNuevas).Error; err == nil && len(facturasNuevas) > 0 {
		log.Printf("Syncing %d new invoice(s) to local...", len(facturasNuevas))
		if err := d.LocalDB.Clauses(clause.OnConflict{DoNothing: true}).Create(&facturasNuevas).Error; err != nil {
			log.Printf("Error syncing Invoices to local: %v", err)
		}
	}

	var ultimaCompraLocal Compra
	d.LocalDB.Order("created_at desc").Limit(1).Find(&ultimaCompraLocal)

	var comprasNuevas []Compra
	queryCompras := d.RemoteDB.Preload("Detalles")
	if !ultimaCompraLocal.CreatedAt.IsZero() {
		queryCompras = queryCompras.Where("created_at > ?", ultimaCompraLocal.CreatedAt)
	}
	if err := queryCompras.Find(&comprasNuevas).Error; err == nil && len(comprasNuevas) > 0 {
		log.Printf("Syncing %d new purchase(s) to local...", len(comprasNuevas))
		if err := d.LocalDB.Clauses(clause.OnConflict{DoNothing: true}).Create(&comprasNuevas).Error; err != nil {
			log.Printf("Error syncing Purchases to local: %v", err)
		}
	}
}

func (d *Db) syncVendedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Vendedor
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("syncVendedorToRemote: no se encontró vendedor local ID %d: %v", id, err)
		return
	}
	d.actualizarLoteEnRemoto(d.RemoteDB, "Vendedores", []interface{}{&record})
}

func (d *Db) syncClienteToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Cliente
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("syncClienteToRemote: no se encontró cliente local ID %d: %v", id, err)
		return
	}
	d.actualizarLoteEnRemoto(d.RemoteDB, "Clientes", []interface{}{&record})
}

func (d *Db) syncProductoToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Producto
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("syncProductoToRemote: no se encontró producto local ID %d: %v", id, err)
		return
	}
	d.actualizarLoteEnRemoto(d.RemoteDB, "Productos", []interface{}{&record})
}

func (d *Db) syncProveedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Proveedor
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("syncProveedorToRemote: no se encontró proveedor local ID %d: %v", id, err)
		return
	}
	d.actualizarLoteEnRemoto(d.RemoteDB, "Proveedores", []interface{}{&record})
}

func (d *Db) syncVentaToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Factura
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Local invoice not found (ID %d): %v", id, err)
		return
	}
	d.syncVendedorToRemote(record.VendedorID)
	d.syncClienteToRemote(record.ClienteID)

	d.ejecutarLote(d.RemoteDB, "Facturas", []interface{}{&record})
}

func (d *Db) syncCompraToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Compra
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Local purchase not found (ID %d): %v", id, err)
		return
	}
	d.syncProveedorToRemote(record.ProveedorID)

	d.ejecutarLote(d.RemoteDB, "Compras", []interface{}{&record})
}

func (d *Db) syncVendedorToLocal(vendedor Vendedor) {
	d.ejecutarLote(d.LocalDB, "Vendedores", []interface{}{&vendedor})
}

func getRecordInfo(record interface{}) RecordInfo {
	switch v := record.(type) {
	case *Vendedor:
		return RecordInfo{Key: v.Cedula, UpdatedAt: v.UpdatedAt, Record: v}
	case *Cliente:
		return RecordInfo{Key: v.NumeroID, UpdatedAt: v.UpdatedAt, Record: v}
	case *Producto:
		return RecordInfo{Key: v.Codigo, UpdatedAt: v.UpdatedAt, Record: v}
	case *Proveedor:
		return RecordInfo{Key: v.Nombre, UpdatedAt: v.UpdatedAt, Record: v}
	default:
		return RecordInfo{}
	}
}

func getCombinedKeys(map1, map2 map[string]RecordInfo) map[string]bool {
	keys := make(map[string]bool)
	for k := range map1 {
		keys[k] = true
	}
	for k := range map2 {
		keys[k] = true
	}
	return keys
}

func sliceToInterfaceSlice(slice interface{}) []interface{} {
	switch s := slice.(type) {
	case *[]Vendedor:
		var iSlice []interface{}
		for i := range *s {
			iSlice = append(iSlice, &(*s)[i])
		}
		return iSlice
	case *[]Cliente:
		var iSlice []interface{}
		for i := range *s {
			iSlice = append(iSlice, &(*s)[i])
		}
		return iSlice
	case *[]Producto:
		var iSlice []interface{}
		for i := range *s {
			iSlice = append(iSlice, &(*s)[i])
		}
		return iSlice
	case *[]Proveedor:
		var iSlice []interface{}
		for i := range *s {
			iSlice = append(iSlice, &(*s)[i])
		}
		return iSlice
	default:
		return nil
	}
}

func getConflictColumn(record interface{}) string {
	switch record.(type) {
	case *Vendedor:
		return "cedula"
	case *Cliente:
		return "numero_id"
	case *Producto:
		return "codigo"
	case *Proveedor:
		return "nombre"
	default:
		return ""
	}
}

func (d *Db) calcularStockRealLocal(productoID uint) int {
	var suma int64
	d.LocalDB.Model(&OperacionStock{}).Where("producto_id = ?", productoID).
		Select("COALESCE(SUM(cantidad_cambio),0)").Scan(&suma)

	if suma == 0 {
		var prod Producto
		if err := d.LocalDB.First(&prod, productoID).Error; err == nil {
			return prod.Stock
		}
	}
	return int(suma)
}
