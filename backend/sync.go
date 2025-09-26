package backend

import (
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecordInfo struct {
	Key       string
	UpdatedAt time.Time
	Record    interface{}
}

// RealizarSincronizacionInicial runs the intelligent synchronization when the application starts.
func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

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

	d.SincronizarOperacionesStock()

	d.sincronizarTransaccionesHaciaLocal()

	log.Println("[END]: Intelligent Synchronization")
}

func (d *Db) sincronizarModelo(modeloSlice interface{}, nombreModelo string) {
	log.Printf("--- Syncing model: %s ---\n", nombreModelo)

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

func (d *Db) ejecutarLote(db *gorm.DB, nombreModelo string, lote []interface{}) {
	if len(lote) == 0 {
		return
	}

	tx := db.Begin()
	if tx.Error != nil {
		log.Printf("Error starting transaction for %s: %v", nombreModelo, tx.Error)
		return
	}

	for _, record := range lote {
		if err := tx.Save(record).Error; err != nil {
			log.Printf("Error saving record for %s during batch: %v", nombreModelo, err)
			tx.Rollback() // Rollback on any error
			return
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
