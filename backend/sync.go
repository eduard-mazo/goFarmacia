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

	d.SincronizarOperacionesStock()

	d.sincronizarTransaccionesHaciaLocal()

	d.Log.Infof("[FIN]: Sincronización Inteligente")
}

func (d *Db) sincronizarModelo(modeloSlice interface{}, nombreModelo string) {
	d.Log.Infof("[SINCRONIZANDO]:\t%s\n", nombreModelo)

	if err := d.LocalDB.Find(modeloSlice).Error; err != nil {
		d.Log.Errorf("Cargando %s desde LOCAL: %v", nombreModelo, err)
		return
	}
	localRecords := sliceToInterfaceSlice(modeloSlice)
	if localRecords == nil {
		d.Log.Warnf("Sin registros en LOCAL para %s", nombreModelo)
	}

	if err := d.RemoteDB.Find(modeloSlice).Error; err != nil {
		d.Log.Errorf("Cargando %s desde REMOTO: %v", nombreModelo, err)
		return
	}
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

	var paraActualizarEnRemoto, paraActualizarEnLocal []interface{}
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
			paraActualizarEnRemoto = append(paraActualizarEnRemoto, local.Record)
		} else if !localExists && remoteExists {
			paraActualizarEnLocal = append(paraActualizarEnLocal, remote.Record)
		}
	}

	if len(paraActualizarEnRemoto) > 0 {
		d.Log.Infof("[%s]\tSincronizando\t%d registro(s) a REMOTO...", nombreModelo, len(paraActualizarEnRemoto))
		d.ejecutarLote(d.RemoteDB, nombreModelo, paraActualizarEnRemoto)
	}

	if len(paraActualizarEnLocal) > 0 {
		d.Log.Infof("[%s]\tSincronizando\t%d registro(s) a LOCAL...", nombreModelo, len(paraActualizarEnLocal))
		d.ejecutarLote(d.LocalDB, nombreModelo, paraActualizarEnLocal)
	}

	if len(paraActualizarEnLocal) == 0 && len(paraActualizarEnRemoto) == 0 {
		d.Log.Infof("[%s]\tNo se encuentran diferencias... MODELO sincronizado", nombreModelo)
	}
}

// ejecutarLote: simple, imprime claves a sincronizar, usa OnConflict UpdateAll y continúa si algún registro falla.
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
		if err := tx.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(record).Error; err != nil {
			log.Printf("Error saving record for %s during batch: %v (continuando)", nombreModelo, err)
			continue
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction for %s: %v", nombreModelo, err)
	}
}

func (d *Db) SincronizarOperacionesStock() {
	if !d.isRemoteDBAvailable() {
		return
	}

	var ops []OperacionStock
	if err := d.LocalDB.Where("sincronizado = ?", false).Find(&ops).Error; err != nil {
		log.Printf("Error loading pending operaciones de stock: %v", err)
		return
	}
	if len(ops) == 0 {
		return
	}

	uniqueMap := make(map[string]OperacionStock)
	for _, op := range ops {
		existing, ok := uniqueMap[op.UUID]
		if !ok || op.Timestamp.After(existing.Timestamp) {
			uniqueMap[op.UUID] = op
		}
	}
	finalOps := make([]OperacionStock, 0, len(uniqueMap))
	for _, op := range uniqueMap {
		finalOps = append(finalOps, op)
	}

	log.Printf("Enviando %d operación(es) de stock al servidor remoto...", len(finalOps))

	for i := range finalOps {
		finalOps[i].ID = 0
	}

	tx := d.RemoteDB.Begin()
	if tx.Error != nil {
		log.Printf("Error iniciando transacción en remoto para operaciones de stock: %v", tx.Error)
		return
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		UpdateAll: true,
	}).Create(&finalOps).Error; err != nil {
		log.Printf("Error en batch upsert operaciones de stock: %v. Realizando fallback por registro...", err)
		tx.Rollback()

		var succeededUUIDs []string
		for _, op := range finalOps {
			op.ID = 0 // asegurar
			if err2 := d.RemoteDB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "uuid"}},
				UpdateAll: true,
			}).Create(&op).Error; err2 != nil {
				log.Printf("Error sincronizando operacion uuid=%s: %v", op.UUID, err2)
				continue
			}
			succeededUUIDs = append(succeededUUIDs, op.UUID)
		}

		if len(succeededUUIDs) > 0 {
			if err3 := d.LocalDB.Model(&OperacionStock{}).
				Where("uuid IN ?", succeededUUIDs).
				Update("sincronizado", true).Error; err3 != nil {
				log.Printf("Error marcando operaciones sincronizadas en local (fallback): %v", err3)
			}
		}
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error confirmando transacción en remoto para operaciones de stock: %v", err)
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
	d.ejecutarLote(d.RemoteDB, "Vendedores", []interface{}{&record})
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
	d.ejecutarLote(d.RemoteDB, "Clientes", []interface{}{&record})
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
	d.ejecutarLote(d.RemoteDB, "Productos", []interface{}{&record})
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
	d.ejecutarLote(d.RemoteDB, "Proveedores", []interface{}{&record})
}

// syncVentaToRemote sincroniza una factura (venta) por ID; asegura vendedor y cliente en remoto.
func (d *Db) syncVentaToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Factura
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Local invoice not found (ID %d): %v", id, err)
		return
	}
	// Asegurar existencia de referencias en remoto
	d.syncVendedorToRemote(record.VendedorID)
	d.syncClienteToRemote(record.ClienteID)

	// Insertar/Actualizar factura con upsert
	d.ejecutarLote(d.RemoteDB, "Facturas", []interface{}{&record})
}

// syncCompraToRemote sincroniza una compra por ID; asegura proveedor en remoto.
func (d *Db) syncCompraToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Compra
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Local purchase not found (ID %d): %v", id, err)
		return
	}
	// Asegurar proveedor en remoto
	d.syncProveedorToRemote(record.ProveedorID)

	// Insertar/Actualizar compra con upsert
	d.ejecutarLote(d.RemoteDB, "Compras", []interface{}{&record})
}

// syncVendedorToLocal: inserta/actualiza vendedor en la DB local (usado por sincronización remota -> local)
func (d *Db) syncVendedorToLocal(vendedor *Vendedor) {
	d.ejecutarLote(d.LocalDB, "Vendedores", []interface{}{vendedor})
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
