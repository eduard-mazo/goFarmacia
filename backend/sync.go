package backend

import (
	"log"

	"gorm.io/gorm/clause"
)

// RealizarSincronizacionInicial es el orquestador que se ejecuta al inicio.
// Decide si es necesario sincronizar y en qué dirección.
func (d *Db) RealizarSincronizacionInicial() {
	if !d.isRemoteDBAvailable() {
		log.Println("Modo offline. Omitiendo sincronización inicial.")
		return
	}

	var localCount, remoteCount int64
	// Usamos la tabla de Vendedores como referencia para ver si hay datos.
	d.LocalDB.Model(&Vendedor{}).Count(&localCount)
	d.RemoteDB.Model(&Vendedor{}).Count(&remoteCount)

	if localCount > 0 && remoteCount == 0 {
		// Caso 1: La base de datos local tiene datos y la remota está vacía.
		log.Println("Detectado: Datos locales existen y la base de datos remota está vacía. Iniciando subida de datos...")
		go d.SincronizarHaciaRemoto() // Se ejecuta en una goroutine para no bloquear el inicio.
	} else if localCount == 0 && remoteCount > 0 {
		// Caso 2: La base de datos local está vacía y la remota tiene datos.
		log.Println("Detectado: Base de datos local vacía y datos remotos existen. Iniciando descarga de datos...")
		go d.SincronizarHaciaLocal() // Se ejecuta en una goroutine.
	} else if localCount == 0 && remoteCount == 0 {
		log.Println("Ambas bases de datos están vacías. No se requiere sincronización inicial.")
	} else {
		log.Println("Ambas bases de datos contienen datos. Se asume que están sincronizadas. La sincronización ocurrirá en tiempo real por cada operación.")
	}
}

// SincronizarHaciaRemoto sube todos los datos de la base de datos local (SQLite)
// a la base de datos remota (Supabase/PostgreSQL).
func (d *Db) SincronizarHaciaRemoto() {
	log.Println("INICIO: Sincronización Local -> Remoto")

	// Sincronizar Vendedores
	if err := d.syncModelHaciaRemoto(&[]Vendedor{}, "Vendedores"); err != nil {
		log.Printf("Error sincronizando Vendedores hacia remoto: %v", err)
	}

	// Sincronizar Clientes
	if err := d.syncModelHaciaRemoto(&[]Cliente{}, "Clientes"); err != nil {
		log.Printf("Error sincronizando Clientes hacia remoto: %v", err)
	}

	// Sincronizar Productos
	if err := d.syncModelHaciaRemoto(&[]Producto{}, "Productos"); err != nil {
		log.Printf("Error sincronizando Productos hacia remoto: %v", err)
	}

	// NOTA: Facturas y Compras no se sincronizan en esta dirección.
	// Usualmente, las ventas se generan en un punto y se suben individualmente.
	// La sincronización masiva de transacciones es compleja y propensa a conflictos.
	// Las funciones que ya tenías (ej. syncVendedorToRemote) se encargan de esto.

	log.Println("FIN: Sincronización Local -> Remoto")
}

// SincronizarHaciaLocal descarga todos los datos de la base de datos remota (Supabase)
// a la base de datos local (SQLite).
func (d *Db) SincronizarHaciaLocal() {
	log.Println("INICIO: Sincronización Remoto -> Local")

	// Sincronizar Vendedores
	if err := d.syncModelHaciaLocal(&[]Vendedor{}, "Vendedores"); err != nil {
		log.Printf("Error sincronizando Vendedores hacia local: %v", err)
	}

	// Sincronizar Clientes
	if err := d.syncModelHaciaLocal(&[]Cliente{}, "Clientes"); err != nil {
		log.Printf("Error sincronizando Clientes hacia local: %v", err)
	}

	// Sincronizar Productos
	if err := d.syncModelHaciaLocal(&[]Producto{}, "Productos"); err != nil {
		log.Printf("Error sincronizando Productos hacia local: %v", err)
	}

	// Sincronizar Facturas y sus detalles (descargar historial)
	var facturas []Factura
	if err := d.RemoteDB.Preload("Detalles").Find(&facturas).Error; err == nil && len(facturas) > 0 {
		if err := d.LocalDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_factura"}}, DoNothing: true}).Create(&facturas).Error; err != nil {
			log.Printf("Error sincronizando Facturas hacia local: %v", err)
		} else {
			log.Printf("Éxito: Sincronizados %d registros de Facturas hacia local.", len(facturas))
		}
	}

	log.Println("FIN: Sincronización Remoto -> Local")
}

// syncModelHaciaRemoto es una función genérica para subir datos de un modelo específico.
func (d *Db) syncModelHaciaRemoto(modelo interface{}, nombreModelo string) error {
	if err := d.LocalDB.Find(modelo).Error; err != nil {
		return err
	}

	// El tipo `reflect.ValueOf(modelo).Elem().Len()` se usa para contar los elementos
	// de forma genérica sin conocer el tipo exacto del slice.
	// if reflect.ValueOf(modelo).Elem().Len() > 0 {
	// La cláusula OnConflict se basa en las restricciones 'unique' de tus structs.
	// Asegúrate de que coincidan con la base de datos remota.
	return d.RemoteDB.Clauses(clause.OnConflict{
		// Columnas con constraint UNIQUE
		Columns: getUniqueColumns(nombreModelo),
		// Columnas a actualizar en caso de conflicto
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}).Create(modelo).Error
	// }
	//return nil
}

// syncModelHaciaLocal es una función genérica para descargar datos de un modelo específico.
func (d *Db) syncModelHaciaLocal(modelo interface{}, nombreModelo string) error {
	// 1. Obtiene todos los registros del remoto (esto está bien)
	if err := d.RemoteDB.Find(modelo).Error; err != nil {
		return err
	}

	// 2. Inserta los registros en la base de datos local en lotes de 500.
	// Esta es la corrección clave para evitar el error en SQLite.
	return d.LocalDB.Clauses(clause.OnConflict{
		Columns:   getUniqueColumns(nombreModelo),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(nombreModelo)),
	}).CreateInBatches(modelo, 500).Error
}

// getUniqueColumns devuelve las columnas con restricción UNIQUE para cada modelo.
func getUniqueColumns(modelName string) []clause.Column {
	switch modelName {
	case "Vendedores":
		return []clause.Column{{Name: "cedula"}}
	case "Clientes":
		return []clause.Column{{Name: "numero_id"}}
	case "Productos":
		return []clause.Column{{Name: "codigo"}}
	default:
		return nil
	}
}

// getUpdatableColumns devuelve las columnas que deben actualizarse en un conflicto.
func getUpdatableColumns(modelName string) []string {
	switch modelName {
	case "Vendedores":
		return []string{"nombre", "apellido", "email", "contrasena", "updated_at"}
	case "Clientes":
		return []string{"nombre", "apellido", "tipo_id", "telefono", "email", "direccion", "updated_at"}
	case "Productos":
		return []string{"nombre", "precio_venta", "stock", "updated_at"}
	default:
		return nil
	}
}
