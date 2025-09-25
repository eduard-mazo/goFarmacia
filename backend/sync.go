package backend

import (
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- PUNTO DE ENTRADA PRINCIPAL PARA LA SINCRONIZACIÓN ---

// RealizarSincronizacionInicial se ejecuta al iniciar la aplicación para reconciliar los datos.
func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

// --- LÓGICA DE SINCRONIZACIÓN INTELIGENTE (BIDIRECCIONAL) ---

// SincronizacionInteligente ejecuta una reconciliación profunda comparando timestamps.
func (d *Db) SincronizacionInteligente() {
	if !d.isRemoteDBAvailable() {
		log.Println("Modo offline. Omitiendo sincronización inteligente.")
		return
	}
	log.Println("[INICIO]: Sincronización Inteligente")

	// 1. Sincronizar modelos no transaccionales (Vendedores, Clientes, etc.)
	d.sincronizarModelo(&[]Vendedor{}, "Vendedores")
	d.sincronizarModelo(&[]Cliente{}, "Clientes")
	d.sincronizarModelo(&[]Proveedor{}, "Proveedores")

	// 2. Sincronizar operaciones de stock pendientes
	d.SincronizarOperacionesStock()

	// 3. Descargar datos transaccionales solo de remoto a local (Ventas, Compras)
	d.sincronizarTransaccionesHaciaLocal()

	log.Println("[FIN]: Sincronización Inteligente")
}

// sincronizarModelo es el algoritmo de reconciliación para un modelo de datos específico.
func (d *Db) sincronizarModelo(modeloSlice interface{}, nombreModelo string) {
	log.Printf("--- Sincronizando modelo: %s ---\n", nombreModelo)

	// 1. Cargar registros de ambas bases de datos
	localRecords, err := d.cargarRegistros(d.LocalDB, modeloSlice)
	if err != nil {
		log.Printf("Error cargando %s desde local: %v", nombreModelo, err)
		return
	}
	remoteRecords, err := d.cargarRegistros(d.RemoteDB, modeloSlice)
	if err != nil {
		log.Printf("Error cargando %s desde remoto: %v", nombreModelo, err)
		return
	}

	// 2. Mapear registros por clave única
	localMap := crearMapaDeRegistros(localRecords)
	remoteMap := crearMapaDeRegistros(remoteRecords)

	// 3. Comparar y decidir qué sincronizar
	var paraActualizarEnRemoto, paraActualizarEnLocal []interface{}
	for key := range getCombinedKeys(localMap, remoteMap) {
		local, localExists := localMap[key]
		remote, remoteExists := remoteMap[key]

		if localExists && remoteExists {
			if local.UpdatedAt.After(remote.UpdatedAt) {
				paraActualizarEnRemoto = append(paraActualizarEnRemoto, local.Record)
			} else if remote.UpdatedAt.After(local.UpdatedAt) {
				paraActualizarEnLocal = append(paraActualizarEnLocal, remote.Record)
			}
		} else if localExists && !remoteExists {
			paraActualizarEnRemoto = append(paraActualizarEnRemoto, local.Record)
		} else if !localExists && remoteExists {
			paraActualizarEnLocal = append(paraActualizarEnLocal, remote.Record)
		}
	}

	// 4. Ejecutar actualizaciones en lotes
	if len(paraActualizarEnRemoto) > 0 {
		log.Printf("[%s] Sincronizando %d registro(s) hacia Remoto...", nombreModelo, len(paraActualizarEnRemoto))
		d.ejecutarLote(d.RemoteDB, nombreModelo, paraActualizarEnRemoto)
	}
	if len(paraActualizarEnLocal) > 0 {
		log.Printf("[%s] Sincronizando %d registro(s) hacia Local...", nombreModelo, len(paraActualizarEnLocal))
		d.ejecutarLote(d.LocalDB, nombreModelo, paraActualizarEnLocal)
	}
}

// --- LÓGICA DE SINCRONIZACIÓN INDIVIDUAL (UNIDIRECCIONAL) ---

// Estas funciones son llamadas después de una acción específica (ej. crear un nuevo cliente).

func (d *Db) syncVendedorToRemote(id uint) {
	var record Vendedor
	d.sincronizarRegistroIndividual(d.LocalDB.Unscoped().First(&record, id), "Vendedor", id)
}
func (d *Db) syncClienteToRemote(id uint) {
	var record Cliente
	d.sincronizarRegistroIndividual(d.LocalDB.Unscoped().First(&record, id), "Cliente", id)
}
func (d *Db) syncProductoToRemote(id uint) {
	var record Producto
	d.sincronizarRegistroIndividual(d.LocalDB.Unscoped().First(&record, id), "Producto", id)
}
func (d *Db) syncProveedorToRemote(id uint) {
	var record Proveedor
	d.sincronizarRegistroIndividual(d.LocalDB.Unscoped().First(&record, id), "Proveedor", id)
}
func (d *Db) syncVentaToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Factura
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Factura local no encontrada (ID %d): %v", id, err)
		return
	}
	d.syncVendedorToRemote(record.VendedorID)
	d.syncClienteToRemote(record.ClienteID)
	if err := d.RemoteDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_factura"}}, DoNothing: true}).Create(&record).Error; err != nil {
		log.Printf("SYNC FAILED for Factura ID %d: %v", id, err)
	}
}
func (d *Db) syncCompraToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Compra
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Compra local no encontrada (ID %d): %v", id, err)
		return
	}
	d.syncProveedorToRemote(record.ProveedorID)
	if err := d.RemoteDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "factura_numero"}}, DoNothing: true}).Create(&record).Error; err != nil {
		log.Printf("SYNC FAILED for Compra ID %d: %v", id, err)
	}
}

// syncVendedorToLocal actualiza la caché local después de un login exitoso contra la BD remota.
func (d *Db) syncVendedorToLocal(vendedor Vendedor) {
	err := d.LocalDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cedula"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "email", "contrasena", "updated_at"}),
	}).Create(&vendedor).Error
	if err != nil {
		log.Printf("LOCAL CACHE FAILED for Vendedor ID %d: %v", vendedor.ID, err)
	}
}

// --- SINCRONIZACIÓN DE STOCK ---

func (d *Db) SincronizarOperacionesStock() {
	if !d.isRemoteDBAvailable() {
		return
	}
	var ops []OperacionStock
	if err := d.LocalDB.Where("sincronizado = ?", false).Find(&ops).Error; err != nil || len(ops) == 0 {
		return
	}
	log.Printf("Enviando %d operaciones de stock al servidor remoto...", len(ops))
	if err := d.RemoteDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "uuid"}}, DoNothing: true}).Create(&ops).Error; err != nil {
		log.Printf("Error al sincronizar operaciones de stock: %v", err)
		return
	}
	var uuids []string
	for _, op := range ops {
		uuids = append(uuids, op.UUID)
	}
	d.LocalDB.Model(&OperacionStock{}).Where("uuid IN ?", uuids).Update("sincronizado", true)
}

// --- FUNCIONES DE AYUDA Y UTILIDADES ---

// RecordInfo almacena datos clave para la comparación.
type RecordInfo struct {
	Key       string
	UpdatedAt time.Time
	Record    interface{}
}

func (d *Db) cargarRegistros(db *gorm.DB, modeloSlice interface{}) ([]interface{}, error) {
	if err := db.Find(modeloSlice).Error; err != nil {
		return nil, err
	}
	return sliceToInterfaceSlice(modeloSlice), nil
}

func crearMapaDeRegistros(records []interface{}) map[string]RecordInfo {
	recordMap := make(map[string]RecordInfo)
	for _, rec := range records {
		info := getRecordInfo(rec)
		if info.Key != "" {
			recordMap[info.Key] = info
		}
	}
	return recordMap
}

func (d *Db) sincronizarRegistroIndividual(result *gorm.DB, nombreModelo string, id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	if result.Error != nil {
		log.Printf("SYNC ERROR: Registro local de %s no encontrado (ID %d): %v", nombreModelo, id, result.Error)
		return
	}
	if err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}).Create(result.Statement.Dest).Error; err != nil {
		log.Printf("SYNC FAILED for %s ID %d: %v", nombreModelo, id, err)
	}
}

func (d *Db) sincronizarTransaccionesHaciaLocal() {
	var facturas []Factura
	if err := d.RemoteDB.Preload("Detalles").Find(&facturas).Error; err == nil && len(facturas) > 0 {
		d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_factura"}}, DoNothing: true}).Create(&facturas)
	}
	var compras []Compra
	if err := d.RemoteDB.Preload("Detalles").Find(&compras).Error; err == nil && len(compras) > 0 {
		d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "factura_numero"}}, DoNothing: true}).Create(&compras)
	}
}

func (d *Db) ejecutarLote(db *gorm.DB, nombreModelo string, lote []interface{}) {
	clauses := clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}
	if err := db.Clauses(clauses).CreateInBatches(lote, 100).Error; err != nil {
		log.Printf("Error al ejecutar lote para %s: %v", nombreModelo, err)
	}
}

func getRecordInfo(record interface{}) RecordInfo {
	switch v := record.(type) {
	case Vendedor:
		return RecordInfo{Key: v.Cedula, UpdatedAt: v.UpdatedAt, Record: v}
	case *Vendedor:
		return RecordInfo{Key: v.Cedula, UpdatedAt: v.UpdatedAt, Record: *v}
	case Cliente:
		return RecordInfo{Key: v.NumeroID, UpdatedAt: v.UpdatedAt, Record: v}
	case *Cliente:
		return RecordInfo{Key: v.NumeroID, UpdatedAt: v.UpdatedAt, Record: *v}
	case Producto:
		return RecordInfo{Key: v.Codigo, UpdatedAt: v.UpdatedAt, Record: v}
	case *Producto:
		return RecordInfo{Key: v.Codigo, UpdatedAt: v.UpdatedAt, Record: *v}
	case Proveedor:
		return RecordInfo{Key: v.Nombre, UpdatedAt: v.UpdatedAt, Record: v}
	case *Proveedor:
		return RecordInfo{Key: v.Nombre, UpdatedAt: v.UpdatedAt, Record: *v}
	}
	return RecordInfo{}
}

func sliceToInterfaceSlice(slice interface{}) []interface{} {
	switch s := slice.(type) {
	case *[]Vendedor:
		var iSlice []interface{}
		for _, v := range *s {
			iSlice = append(iSlice, v)
		}
		return iSlice
	case *[]Cliente:
		var iSlice []interface{}
		for _, v := range *s {
			iSlice = append(iSlice, v)
		}
		return iSlice
	case *[]Producto:
		var iSlice []interface{}
		for _, v := range *s {
			iSlice = append(iSlice, v)
		}
		return iSlice
	case *[]Proveedor:
		var iSlice []interface{}
		for _, v := range *s {
			iSlice = append(iSlice, v)
		}
		return iSlice
	}
	return nil
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

func getUniqueColumns(modelName string) []clause.Column {
	switch modelName {
	case "Vendedor", "Vendedores":
		return []clause.Column{{Name: "cedula"}}
	case "Cliente", "Clientes":
		return []clause.Column{{Name: "numero_id"}}
	case "Producto", "Productos":
		return []clause.Column{{Name: "codigo"}}
	case "Proveedor", "Proveedores":
		return []clause.Column{{Name: "nombre"}}
	}
	return nil
}

func getUpdatableColumns(modelName string) []string {
	switch modelName {
	case "Vendedor", "Vendedores":
		return []string{"nombre", "apellido", "email", "contrasena", "updated_at", "deleted_at"}
	case "Cliente", "Clientes":
		return []string{"nombre", "apellido", "tipo_id", "telefono", "email", "direccion", "updated_at", "deleted_at"}
	case "Producto", "Productos":
		return []string{"nombre", "precio_venta", "stock", "updated_at", "deleted_at"}
	case "Proveedor", "Proveedores":
		return []string{"telefono", "email", "updated_at", "deleted_at"}
	}
	return nil
}
