package backend

import (
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecordInfo is a struct to hold key record information for comparison.
type RecordInfo struct {
	Key       string
	UpdatedAt time.Time
	Record    interface{}
}

// RealizarSincronizacionInicial runs the intelligent synchronization when the application starts.
func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

// SincronizacionInteligente performs a deep reconciliation between local and remote databases.
func (d *Db) SincronizacionInteligente() {
	if !d.isRemoteDBAvailable() {
		log.Println("Offline mode. Skipping intelligent synchronization.")
		return
	}
	log.Println("[START]: Intelligent Synchronization")

	d.sincronizarModelo(&[]Vendedor{}, "Vendedores")
	d.sincronizarModelo(&[]Cliente{}, "Clientes")
	d.sincronizarModelo(&[]Proveedor{}, "Proveedores")
	d.sincronizarModelo(&[]Producto{}, "Productos")

	// Sync new stock operations
	d.SincronizarOperacionesStock()

	// Sync transactions (Invoices, Purchases) from remote to local
	d.sincronizarTransaccionesHaciaLocal()

	log.Println("[END]: Intelligent Synchronization")
}

// sincronizarModelo is the core of the reconciliation algorithm for a specific data model.
func (d *Db) sincronizarModelo(modeloSlice interface{}, nombreModelo string) {
	log.Printf("--- Syncing model: %s ---\n", nombreModelo)

	// 1. Load all records from both databases
	var localRecords, remoteRecords []interface{}
	if err := d.LocalDB.Find(modeloSlice).Error; err != nil {
		log.Printf("Error loading %s from local: %v", nombreModelo, err)
		return
	}
	localRecords = sliceToInterfaceSlice(modeloSlice)

	if err := d.RemoteDB.Find(modeloSlice).Error; err != nil {
		log.Printf("Error loading %s from remote: %v", nombreModelo, err)
		return
	}
	remoteRecords = sliceToInterfaceSlice(modeloSlice)

	// 2. Map records by their unique business key
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

	// 3. Compare and decide what to sync
	var paraActualizarEnRemoto, paraActualizarEnLocal []interface{}
	allKeys := getCombinedKeys(localMap, remoteMap)

	for key := range allKeys {
		local, localExists := localMap[key]
		remote, remoteExists := remoteMap[key]

		if localExists && remoteExists {
			// Truncate to millisecond precision to avoid timezone/db precision issues
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

	// 4. Execute batch updates using the helper function
	if len(paraActualizarEnRemoto) > 0 {
		log.Printf("[%s] Syncing %d record(s) to Remote...", nombreModelo, len(paraActualizarEnRemoto))
		d.ejecutarLote(d.RemoteDB, nombreModelo, paraActualizarEnRemoto)
	}

	if len(paraActualizarEnLocal) > 0 {
		log.Printf("[%s] Syncing %d record(s) to Local...", nombreModelo, len(paraActualizarEnLocal))
		d.ejecutarLote(d.LocalDB, nombreModelo, paraActualizarEnLocal)
	}

	if len(paraActualizarEnLocal) == 0 && len(paraActualizarEnRemoto) == 0 {
		log.Printf("[%s] No differences found. Model already synchronized.", nombreModelo)
	}
}

// ejecutarLote converts the generic slice to a typed slice and safely executes the batch operation.
func (d *Db) ejecutarLote(db *gorm.DB, nombreModelo string, lote []interface{}) {
	if len(lote) == 0 {
		return
	}

	clauses := clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}

	var typedBatchToInsert interface{}
	switch nombreModelo {
	case "Vendedores":
		records := make([]Vendedor, len(lote))
		for i, item := range lote {
			records[i] = item.(Vendedor)
		}
		typedBatchToInsert = records
	case "Clientes":
		records := make([]Cliente, len(lote))
		for i, item := range lote {
			records[i] = item.(Cliente)
		}
		typedBatchToInsert = records
	case "Productos":
		records := make([]Producto, len(lote))
		for i, item := range lote {
			records[i] = item.(Producto)
		}
		typedBatchToInsert = records
	case "Proveedores":
		records := make([]Proveedor, len(lote))
		for i, item := range lote {
			records[i] = item.(Proveedor)
		}
		typedBatchToInsert = records
	default:
		log.Printf("Unknown model '%s' in ejecutarLote, cannot process batch.", nombreModelo)
		return
	}

	err := db.Clauses(clauses).CreateInBatches(typedBatchToInsert, 100).Error

	if err != nil {
		log.Printf("Error executing batch for %s: %v", nombreModelo, err)
	}
}

// SincronizarOperacionesStock sends pending stock operations to the remote DB.
func (d *Db) SincronizarOperacionesStock() {
	if !d.isRemoteDBAvailable() {
		return
	}
	var ops []OperacionStock
	if err := d.LocalDB.Where("sincronizado = ?", false).Find(&ops).Error; err != nil || len(ops) == 0 {
		return
	}
	log.Printf("Sending %d stock operations to remote server...", len(ops))
	if err := d.RemoteDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "uuid"}}, DoNothing: true}).Create(&ops).Error; err != nil {
		log.Printf("Error syncing stock operations: %v", err)
		return
	}
	var uuids []string
	for _, op := range ops {
		uuids = append(uuids, op.UUID)
	}
	d.LocalDB.Model(&OperacionStock{}).Where("uuid IN ?", uuids).Update("sincronizado", true)
}

// sincronizarTransaccionesHaciaLocal downloads read-only transactions (Invoices, Purchases) to the local DB.
func (d *Db) sincronizarTransaccionesHaciaLocal() {
	var facturas []Factura
	if err := d.RemoteDB.Preload("Detalles").Find(&facturas).Error; err == nil && len(facturas) > 0 {
		if err := d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_factura"}}, DoNothing: true}).Create(&facturas).Error; err != nil {
			log.Printf("Error syncing Invoices to local: %v", err)
		}
	}

	var compras []Compra
	if err := d.RemoteDB.Preload("Detalles").Find(&compras).Error; err == nil && len(compras) > 0 {
		if err := d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "factura_numero"}}, DoNothing: true}).Create(&compras).Error; err != nil {
			log.Printf("Error syncing Purchases to local: %v", err)
		}
	}
}

// --- SINGLE-ACTION SYNC FUNCTIONS ---
// These are called immediately after a specific action (e.g., creating a new client).

func (d *Db) syncVendedorToRemote(id uint) {
	var record Vendedor
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err == nil {
		d.ejecutarLote(d.RemoteDB, "Vendedores", []interface{}{record})
	}
}
func (d *Db) syncClienteToRemote(id uint) {
	var record Cliente
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err == nil {
		d.ejecutarLote(d.RemoteDB, "Clientes", []interface{}{record})
	}
}
func (d *Db) syncProductoToRemote(id uint) {
	var record Producto
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err == nil {
		d.ejecutarLote(d.RemoteDB, "Productos", []interface{}{record})
	}
}
func (d *Db) syncProveedorToRemote(id uint) {
	var record Proveedor
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err == nil {
		d.ejecutarLote(d.RemoteDB, "Proveedores", []interface{}{record})
	}
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
	if err := d.RemoteDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_factura"}}, DoNothing: true}).Create(&record).Error; err != nil {
		log.Printf("SYNC FAILED for Invoice ID %d: %v", id, err)
	}
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
	if err := d.RemoteDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "factura_numero"}}, DoNothing: true}).Create(&record).Error; err != nil {
		log.Printf("SYNC FAILED for Purchase ID %d: %v", id, err)
	}
}
func (d *Db) syncVendedorToLocal(vendedor Vendedor) {
	d.ejecutarLote(d.LocalDB, "Vendedores", []interface{}{vendedor})
}

// --- HELPER FUNCTIONS ---

func getRecordInfo(record interface{}) RecordInfo {
	switch v := record.(type) {
	case Vendedor:
		return RecordInfo{Key: v.Cedula, UpdatedAt: v.UpdatedAt, Record: v}
	case Cliente:
		return RecordInfo{Key: v.NumeroID, UpdatedAt: v.UpdatedAt, Record: v}
	case Producto:
		return RecordInfo{Key: v.Codigo, UpdatedAt: v.UpdatedAt, Record: v}
	case Proveedor:
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
	default:
		return nil
	}
}

func getUniqueColumns(modelName string) []clause.Column {
	switch modelName {
	case "Vendedores":
		return []clause.Column{{Name: "cedula"}}
	case "Clientes":
		return []clause.Column{{Name: "numero_id"}}
	case "Productos":
		return []clause.Column{{Name: "codigo"}}
	case "Proveedores":
		return []clause.Column{{Name: "nombre"}}
	default:
		return nil
	}
}

func getUpdatableColumns(modelName string) []string {
	switch modelName {
	case "Vendedores":
		return []string{"nombre", "apellido", "email", "contrasena", "updated_at", "deleted_at"}
	case "Clientes":
		return []string{"nombre", "apellido", "tipo_id", "telefono", "email", "direccion", "updated_at", "deleted_at"}
	case "Productos":
		return []string{"nombre", "precio_venta", "stock", "updated_at", "deleted_at"}
	case "Proveedores":
		return []string{"telefono", "email", "updated_at", "deleted_at"}
	default:
		return nil
	}
}
