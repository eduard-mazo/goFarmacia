package backend

import (
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecordInfo es una estructura para almacenar la información clave de un registro para la comparación.
type RecordInfo struct {
	Key       string
	UpdatedAt time.Time
	Record    interface{}
}

// SincronizacionInteligente ejecuta una reconciliación profunda entre la BD local y remota.
func (d *Db) SincronizacionInteligente() {
	if !d.isRemoteDBAvailable() {
		log.Println("Modo offline. Omitiendo sincronización inteligente.")
		return
	}
	log.Println("INICIO: Sincronización Inteligente")
	d.sincronizarModelo(&[]Vendedor{}, "Vendedores")
	d.sincronizarModelo(&[]Cliente{}, "Clientes")
	d.sincronizarModelo(&[]Producto{}, "Productos")
	d.sincronizarModelo(&[]Proveedor{}, "Proveedores")
	d.SincronizarHaciaLocal() // Sincroniza facturas y compras que no se reconcilian.
	log.Println("FIN: Sincronización Inteligente")
}

// sincronizarModelo es el corazón del algoritmo de reconciliación para un modelo de datos específico.
func (d *Db) sincronizarModelo(modeloSlice interface{}, nombreModelo string) {
	log.Printf("--- Sincronizando modelo: %s ---\n", nombreModelo)

	// 1. Cargar todos los registros de ambas bases de datos
	var localRecords, remoteRecords []interface{}
	if err := d.LocalDB.Find(modeloSlice).Error; err != nil {
		log.Printf("Error cargando %s desde local: %v", nombreModelo, err)
		return
	}
	localRecords = sliceToInterfaceSlice(modeloSlice)

	if err := d.RemoteDB.Find(modeloSlice).Error; err != nil {
		log.Printf("Error cargando %s desde remoto: %v", nombreModelo, err)
		return
	}
	remoteRecords = sliceToInterfaceSlice(modeloSlice)

	// 2. Mapear registros por su clave única de negocio
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

	// 3. Comparar y decidir qué sincronizar
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

	// 4. Ejecutar actualizaciones en lotes usando la función de ayuda
	if len(paraActualizarEnRemoto) > 0 {
		log.Printf("[%s] Sincronizando %d registro(s) hacia Remoto...", nombreModelo, len(paraActualizarEnRemoto))
		d.ejecutarLote(d.RemoteDB, nombreModelo, paraActualizarEnRemoto)
	}

	if len(paraActualizarEnLocal) > 0 {
		log.Printf("[%s] Sincronizando %d registro(s) hacia Local...", nombreModelo, len(paraActualizarEnLocal))
		d.ejecutarLote(d.LocalDB, nombreModelo, paraActualizarEnLocal)
	}

	if len(paraActualizarEnLocal) == 0 && len(paraActualizarEnRemoto) == 0 {
		log.Printf("[%s] No se encontraron diferencias. Modelo ya sincronizado.", nombreModelo)
	}
}

// ejecutarLote convierte el slice genérico a un slice tipado y ejecuta la operación en lote de forma segura.
func (d *Db) ejecutarLote(db *gorm.DB, nombreModelo string, lote []interface{}) {
	clauses := clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}

	var err error
	switch nombreModelo {
	case "Vendedores":
		var records []Vendedor
		for _, item := range lote {
			records = append(records, item.(Vendedor))
		}
		err = db.Model(&Vendedor{}).Clauses(clauses).CreateInBatches(records, 100).Error
	case "Clientes":
		var records []Cliente
		for _, item := range lote {
			records = append(records, item.(Cliente))
		}
		err = db.Model(&Cliente{}).Clauses(clauses).CreateInBatches(records, 100).Error
	case "Productos":
		var records []Producto
		for _, item := range lote {
			records = append(records, item.(Producto))
		}
		err = db.Model(&Producto{}).Clauses(clauses).CreateInBatches(records, 100).Error
	case "Proveedores":
		var records []Proveedor
		for _, item := range lote {
			records = append(records, item.(Proveedor))
		}
		err = db.Model(&Proveedor{}).Clauses(clauses).CreateInBatches(records, 100).Error
	default:
		log.Printf("Modelo desconocido '%s' en ejecutarLote, no se pudo procesar el lote.", nombreModelo)
		return
	}

	if err != nil {
		log.Printf("Error al ejecutar lote para %s: %v", nombreModelo, err)
	}
}

// --- Funciones de Ayuda para el Algoritmo ---

// getRecordInfo extrae la clave única y el timestamp de un registro genérico.
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

// getCombinedKeys crea un set con todas las claves de ambos mapas para asegurar que se itere sobre todos los registros.
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

// sliceToInterfaceSlice convierte slices tipados (ej. []Producto) a []interface{}
func sliceToInterfaceSlice(slice interface{}) []interface{} {
	switch s := slice.(type) {
	case *[]Vendedor:
		var interfaceSlice []interface{}
		for _, v := range *s {
			interfaceSlice = append(interfaceSlice, v)
		}
		return interfaceSlice
	case *[]Cliente:
		var interfaceSlice []interface{}
		for _, v := range *s {
			interfaceSlice = append(interfaceSlice, v)
		}
		return interfaceSlice
	case *[]Producto:
		var interfaceSlice []interface{}
		for _, v := range *s {
			interfaceSlice = append(interfaceSlice, v)
		}
		return interfaceSlice
	case *[]Proveedor:
		var interfaceSlice []interface{}
		for _, v := range *s {
			interfaceSlice = append(interfaceSlice, v)
		}
		return interfaceSlice
	default:
		return nil
	}
}

// RealizarSincronizacionInicial dispara la nueva sincronización inteligente.
func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

// SincronizarHaciaRemoto sube todos los datos de la local a la remota.
func (d *Db) SincronizarHaciaRemoto() {
	log.Println("INICIO: Sincronización Local -> Remoto")
	d.syncModelHaciaRemoto(&[]Vendedor{}, "Vendedores")
	d.syncModelHaciaRemoto(&[]Cliente{}, "Clientes")
	d.syncModelHaciaRemoto(&[]Producto{}, "Productos")
	d.syncModelHaciaRemoto(&[]Proveedor{}, "Proveedores")
	log.Println("FIN: Sincronización Local -> Remoto")
}

// SincronizarHaciaLocal descarga todos los datos de la remota a la local.
func (d *Db) SincronizarHaciaLocal() {
	log.Println("INICIO: Sincronización Remoto -> Local")

	d.syncModelHaciaLocal(&[]Vendedor{}, "Vendedores")
	d.syncModelHaciaLocal(&[]Cliente{}, "Clientes")
	d.syncModelHaciaLocal(&[]Producto{}, "Productos")
	d.syncModelHaciaLocal(&[]Proveedor{}, "Proveedores")

	var facturas []Factura
	if err := d.RemoteDB.Preload("Detalles").Find(&facturas).Error; err == nil && len(facturas) > 0 {
		if err := d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_factura"}}, DoNothing: true}).Create(&facturas).Error; err != nil {
			log.Printf("Error sincronizando Facturas hacia local: %v", err)
		}
	}

	var compras []Compra
	if err := d.RemoteDB.Preload("Detalles").Find(&compras).Error; err == nil && len(compras) > 0 {
		if err := d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "factura_numero"}}, DoNothing: true}).Create(&compras).Error; err != nil {
			log.Printf("Error sincronizando Compras hacia local: %v", err)
		}
	}

	log.Println("FIN: Sincronización Remoto -> Local")
}

func (d *Db) syncModelHaciaRemoto(modelo interface{}, nombreModelo string) {
	if err := d.LocalDB.Find(modelo).Error; err != nil {
		log.Printf("Error cargando %s desde local: %v", nombreModelo, err)
		return
	}
	if err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}).Create(modelo).Error; err != nil {
		log.Printf("Error sincronizando %s hacia remoto: %v", nombreModelo, err)
	}
}

func (d *Db) syncModelHaciaLocal(modelo interface{}, nombreModelo string) {
	if err := d.RemoteDB.Find(modelo).Error; err != nil {
		log.Printf("Error cargando %s desde remoto: %v", nombreModelo, err)
		return
	}
	if err := d.LocalDB.Clauses(clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}).CreateInBatches(modelo, 500).Error; err != nil {
		log.Printf("Error sincronizando %s hacia local: %v", nombreModelo, err)
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
		return []string{"nombre", "apellido", "email", "contrasena", "updated_at"}
	case "Clientes":
		return []string{"nombre", "apellido", "tipo_id", "telefono", "email", "direccion", "updated_at"}
	case "Productos":
		return []string{"nombre", "precio_venta", "stock", "updated_at"}
	case "Proveedores":
		return []string{"telefono", "email", "updated_at"}
	default:
		return nil
	}
}
