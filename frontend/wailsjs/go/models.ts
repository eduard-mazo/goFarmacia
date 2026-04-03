export namespace backend {
	
	export class AjusteStockRequest {
	    ProductoUUID: string;
	    NuevoStock: number;
	
	    static createFrom(source: any = {}) {
	        return new AjusteStockRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProductoUUID = source["ProductoUUID"];
	        this.NuevoStock = source["NuevoStock"];
	    }
	}
	export class BancolombiaAuthStatus {
	    authenticated: boolean;
	    credPresent: boolean;
	    configDir: string;
	
	    static createFrom(source: any = {}) {
	        return new BancolombiaAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.credPresent = source["credPresent"];
	        this.configDir = source["configDir"];
	    }
	}
	export class BancolombiaAutoPollingState {
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BancolombiaAutoPollingState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	    }
	}
	export class BancolombiaOpcionesPeriodo {
	    modo: string;
	    desde: string;
	    hasta: string;
	
	    static createFrom(source: any = {}) {
	        return new BancolombiaOpcionesPeriodo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modo = source["modo"];
	        this.desde = source["desde"];
	        this.hasta = source["hasta"];
	    }
	}
	export class BancolombiaProgressState {
	    revisados: number;
	    total: number;
	    nuevas: number;
	
	    static createFrom(source: any = {}) {
	        return new BancolombiaProgressState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revisados = source["revisados"];
	        this.total = source["total"];
	        this.nuevas = source["nuevas"];
	    }
	}
	export class Cliente {
	    CreatedAt: string;
	    UpdatedAt: string;
	    DeletedAt?: string;
	    UUID: string;
	    Nombre: string;
	    Apellido: string;
	    TipoID: string;
	    NumeroID: string;
	    Telefono: string;
	    Email: string;
	    Direccion: string;
	
	    static createFrom(source: any = {}) {
	        return new Cliente(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	        this.DeletedAt = source["DeletedAt"];
	        this.UUID = source["UUID"];
	        this.Nombre = source["Nombre"];
	        this.Apellido = source["Apellido"];
	        this.TipoID = source["TipoID"];
	        this.NumeroID = source["NumeroID"];
	        this.Telefono = source["Telefono"];
	        this.Email = source["Email"];
	        this.Direccion = source["Direccion"];
	    }
	}
	export class ColumnInfo {
	    Position: number;
	    Name: string;
	    DataType: string;
	    IsNullable: boolean;
	    Default: string;
	    IsPrimaryKey: boolean;
	    IsForeignKey: boolean;
	    MaxLength: number;
	
	    static createFrom(source: any = {}) {
	        return new ColumnInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Position = source["Position"];
	        this.Name = source["Name"];
	        this.DataType = source["DataType"];
	        this.IsNullable = source["IsNullable"];
	        this.Default = source["Default"];
	        this.IsPrimaryKey = source["IsPrimaryKey"];
	        this.IsForeignKey = source["IsForeignKey"];
	        this.MaxLength = source["MaxLength"];
	    }
	}
	export class DetalleCompra {
	    UUID: string;
	    CompraUUID: string;
	    ProductoUUID: string;
	    Cantidad: number;
	    PrecioCompraUnitario: number;
	
	    static createFrom(source: any = {}) {
	        return new DetalleCompra(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.CompraUUID = source["CompraUUID"];
	        this.ProductoUUID = source["ProductoUUID"];
	        this.Cantidad = source["Cantidad"];
	        this.PrecioCompraUnitario = source["PrecioCompraUnitario"];
	    }
	}
	export class Compra {
	    UUID: string;
	    Fecha: string;
	    ProveedorUUID: string;
	    FacturaNumero: string;
	    Total: number;
	    Detalles: DetalleCompra[];
	
	    static createFrom(source: any = {}) {
	        return new Compra(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.Fecha = source["Fecha"];
	        this.ProveedorUUID = source["ProveedorUUID"];
	        this.FacturaNumero = source["FacturaNumero"];
	        this.Total = source["Total"];
	        this.Detalles = this.convertValues(source["Detalles"], DetalleCompra);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompraProducto {
	    ProductoUUID: string;
	    Cantidad: number;
	    PrecioCompraUnitario: number;
	
	    static createFrom(source: any = {}) {
	        return new CompraProducto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProductoUUID = source["ProductoUUID"];
	        this.Cantidad = source["Cantidad"];
	        this.PrecioCompraUnitario = source["PrecioCompraUnitario"];
	    }
	}
	export class CompraRequest {
	    ProveedorUUID: string;
	    FacturaNumero: string;
	    Productos: CompraProducto[];
	
	    static createFrom(source: any = {}) {
	        return new CompraRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProveedorUUID = source["ProveedorUUID"];
	        this.FacturaNumero = source["FacturaNumero"];
	        this.Productos = this.convertValues(source["Productos"], CompraProducto);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConstraintInfo {
	    Name: string;
	    Type: string;
	    Definition: string;
	
	    static createFrom(source: any = {}) {
	        return new ConstraintInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.Definition = source["Definition"];
	    }
	}
	export class CredencialesInfo {
	    Exists: boolean;
	    Path: string;
	    ProjectID: string;
	    ClientID: string;
	
	    static createFrom(source: any = {}) {
	        return new CredencialesInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Exists = source["Exists"];
	        this.Path = source["Path"];
	        this.ProjectID = source["ProjectID"];
	        this.ClientID = source["ClientID"];
	    }
	}
	export class DBStatusResponse {
	    Connected: boolean;
	    SetupMode: boolean;
	    Message: string;
	    DSNHint: string;
	
	    static createFrom(source: any = {}) {
	        return new DBStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Connected = source["Connected"];
	        this.SetupMode = source["SetupMode"];
	        this.Message = source["Message"];
	        this.DSNHint = source["DSNHint"];
	    }
	}
	export class MetodoPagoDia {
	    metodo_pago: string;
	    count: number;
	    monto: number;
	
	    static createFrom(source: any = {}) {
	        return new MetodoPagoDia(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.metodo_pago = source["metodo_pago"];
	        this.count = source["count"];
	        this.monto = source["monto"];
	    }
	}
	export class VendedorRendimiento {
	    nombreCompleto: string;
	    totalVendido: number;
	    numVentas: number;
	
	    static createFrom(source: any = {}) {
	        return new VendedorRendimiento(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nombreCompleto = source["nombreCompleto"];
	        this.totalVendido = source["totalVendido"];
	        this.numVentas = source["numVentas"];
	    }
	}
	export class Producto {
	    CreatedAt: string;
	    UpdatedAt: string;
	    DeletedAt?: string;
	    UUID: string;
	    Nombre: string;
	    Codigo: string;
	    PrecioVenta: number;
	    Stock: number;
	
	    static createFrom(source: any = {}) {
	        return new Producto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	        this.DeletedAt = source["DeletedAt"];
	        this.UUID = source["UUID"];
	        this.Nombre = source["Nombre"];
	        this.Codigo = source["Codigo"];
	        this.PrecioVenta = source["PrecioVenta"];
	        this.Stock = source["Stock"];
	    }
	}
	export class ProductoVendido {
	    nombre: string;
	    cantidad: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductoVendido(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nombre = source["nombre"];
	        this.cantidad = source["cantidad"];
	    }
	}
	export class VentaIndividual {
	    timestamp: string;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new VentaIndividual(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.total = source["total"];
	    }
	}
	export class DashboardData {
	    totalVentasDia: number;
	    numeroVentasDia: number;
	    ticketPromedioDia: number;
	    ventasIndividuales: VentaIndividual[];
	    topProductos: ProductoVendido[];
	    productosSinStock: Producto[];
	    topVendedor: VendedorRendimiento;
	    topVendedoresDia: VendedorRendimiento[];
	    metodosPago: MetodoPagoDia[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalVentasDia = source["totalVentasDia"];
	        this.numeroVentasDia = source["numeroVentasDia"];
	        this.ticketPromedioDia = source["ticketPromedioDia"];
	        this.ventasIndividuales = this.convertValues(source["ventasIndividuales"], VentaIndividual);
	        this.topProductos = this.convertValues(source["topProductos"], ProductoVendido);
	        this.productosSinStock = this.convertValues(source["productosSinStock"], Producto);
	        this.topVendedor = this.convertValues(source["topVendedor"], VendedorRendimiento);
	        this.topVendedoresDia = this.convertValues(source["topVendedoresDia"], VendedorRendimiento);
	        this.metodosPago = this.convertValues(source["metodosPago"], MetodoPagoDia);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DetalleFactura {
	    CreatedAt: string;
	    UpdatedAt: string;
	    DeletedAt?: string;
	    UUID: string;
	    FacturaUUID: string;
	    ProductoUUID: string;
	    Producto: Producto;
	    Cantidad: number;
	    PrecioUnitario: number;
	    PrecioTotal: number;
	
	    static createFrom(source: any = {}) {
	        return new DetalleFactura(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	        this.DeletedAt = source["DeletedAt"];
	        this.UUID = source["UUID"];
	        this.FacturaUUID = source["FacturaUUID"];
	        this.ProductoUUID = source["ProductoUUID"];
	        this.Producto = this.convertValues(source["Producto"], Producto);
	        this.Cantidad = source["Cantidad"];
	        this.PrecioUnitario = source["PrecioUnitario"];
	        this.PrecioTotal = source["PrecioTotal"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DriveAuthStatus {
	    authenticated: boolean;
	    credPresent: boolean;
	    configDir: string;
	
	    static createFrom(source: any = {}) {
	        return new DriveAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.credPresent = source["credPresent"];
	        this.configDir = source["configDir"];
	    }
	}
	export class DriveAutoBackupState {
	    enabled: boolean;
	    nextBackup: string;
	    lastBackup: string;
	
	    static createFrom(source: any = {}) {
	        return new DriveAutoBackupState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.nextBackup = source["nextBackup"];
	        this.lastBackup = source["lastBackup"];
	    }
	}
	export class DriveBackupFile {
	    id: string;
	    name: string;
	    sizeBytes: number;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DriveBackupFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sizeBytes = source["sizeBytes"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class DriveBackupResult {
	    fileId: string;
	    fileName: string;
	    sizeBytes: number;
	    ts: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DriveBackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileId = source["fileId"];
	        this.fileName = source["fileName"];
	        this.sizeBytes = source["sizeBytes"];
	        this.ts = source["ts"];
	        this.error = source["error"];
	    }
	}
	export class EnriquecerResult {
	    Procesadas: number;
	    Enriquecidas: number;
	    Errores: number;
	
	    static createFrom(source: any = {}) {
	        return new EnriquecerResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Procesadas = source["Procesadas"];
	        this.Enriquecidas = source["Enriquecidas"];
	        this.Errores = source["Errores"];
	    }
	}
	export class Vendedor {
	    CreatedAt: string;
	    UpdatedAt: string;
	    DeletedAt?: string;
	    UUID: string;
	    Nombre: string;
	    Apellido: string;
	    Cedula: string;
	    Email: string;
	    Contrasena: string;
	    MFAEnabled: boolean;
	    Role: string;
	
	    static createFrom(source: any = {}) {
	        return new Vendedor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	        this.DeletedAt = source["DeletedAt"];
	        this.UUID = source["UUID"];
	        this.Nombre = source["Nombre"];
	        this.Apellido = source["Apellido"];
	        this.Cedula = source["Cedula"];
	        this.Email = source["Email"];
	        this.Contrasena = source["Contrasena"];
	        this.MFAEnabled = source["MFAEnabled"];
	        this.Role = source["Role"];
	    }
	}
	export class Factura {
	    CreatedAt: string;
	    UpdatedAt: string;
	    DeletedAt?: string;
	    UUID: string;
	    NumeroFactura: string;
	    FechaEmision: string;
	    VendedorUUID: string;
	    Vendedor: Vendedor;
	    ClienteUUID: string;
	    Cliente: Cliente;
	    Subtotal: number;
	    IVA: number;
	    Total: number;
	    Estado: string;
	    MetodoPago: string;
	    Detalles: DetalleFactura[];
	
	    static createFrom(source: any = {}) {
	        return new Factura(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	        this.DeletedAt = source["DeletedAt"];
	        this.UUID = source["UUID"];
	        this.NumeroFactura = source["NumeroFactura"];
	        this.FechaEmision = source["FechaEmision"];
	        this.VendedorUUID = source["VendedorUUID"];
	        this.Vendedor = this.convertValues(source["Vendedor"], Vendedor);
	        this.ClienteUUID = source["ClienteUUID"];
	        this.Cliente = this.convertValues(source["Cliente"], Cliente);
	        this.Subtotal = source["Subtotal"];
	        this.IVA = source["IVA"];
	        this.Total = source["Total"];
	        this.Estado = source["Estado"];
	        this.MetodoPago = source["MetodoPago"];
	        this.Detalles = this.convertValues(source["Detalles"], DetalleFactura);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FacturaCompraDetalle {
	    UUID: string;
	    FacturaCompraUUID: string;
	    CodigoProducto: string;
	    Descripcion: string;
	    Cantidad: number;
	    PrecioUnitario: number;
	    TotalLinea: number;
	    ImpuestoLinea: number;
	    Propiedades: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new FacturaCompraDetalle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.FacturaCompraUUID = source["FacturaCompraUUID"];
	        this.CodigoProducto = source["CodigoProducto"];
	        this.Descripcion = source["Descripcion"];
	        this.Cantidad = source["Cantidad"];
	        this.PrecioUnitario = source["PrecioUnitario"];
	        this.TotalLinea = source["TotalLinea"];
	        this.ImpuestoLinea = source["ImpuestoLinea"];
	        this.Propiedades = source["Propiedades"];
	    }
	}
	export class FacturaCompra {
	    UUID: string;
	    ProveedorUUID?: string;
	    ProveedorNIT: string;
	    ProveedorNombre: string;
	    ClienteNIT: string;
	    ClienteNombre: string;
	    NumeroFactura: string;
	    CUFE: string;
	    FechaEmision: string;
	    Moneda: string;
	    Subtotal: number;
	    IVA: number;
	    Total: number;
	    Estado: string;
	    TipoDocumento: string;
	    ReferenciaDocumento: string;
	    EmailMessageID: string;
	    Detalles: FacturaCompraDetalle[];
	
	    static createFrom(source: any = {}) {
	        return new FacturaCompra(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.ProveedorUUID = source["ProveedorUUID"];
	        this.ProveedorNIT = source["ProveedorNIT"];
	        this.ProveedorNombre = source["ProveedorNombre"];
	        this.ClienteNIT = source["ClienteNIT"];
	        this.ClienteNombre = source["ClienteNombre"];
	        this.NumeroFactura = source["NumeroFactura"];
	        this.CUFE = source["CUFE"];
	        this.FechaEmision = source["FechaEmision"];
	        this.Moneda = source["Moneda"];
	        this.Subtotal = source["Subtotal"];
	        this.IVA = source["IVA"];
	        this.Total = source["Total"];
	        this.Estado = source["Estado"];
	        this.TipoDocumento = source["TipoDocumento"];
	        this.ReferenciaDocumento = source["ReferenciaDocumento"];
	        this.EmailMessageID = source["EmailMessageID"];
	        this.Detalles = this.convertValues(source["Detalles"], FacturaCompraDetalle);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FacturaVentaResumen {
	    uuid: string;
	    numeroFactura: string;
	    clienteNombre: string;
	    total: number;
	    fecha: string;
	
	    static createFrom(source: any = {}) {
	        return new FacturaVentaResumen(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.numeroFactura = source["numeroFactura"];
	        this.clienteNombre = source["clienteNombre"];
	        this.total = source["total"];
	        this.fecha = source["fecha"];
	    }
	}
	export class FacturasCompraResponse {
	    Records: FacturaCompra[];
	    TotalRecords: number;
	
	    static createFrom(source: any = {}) {
	        return new FacturasCompraResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Records = this.convertValues(source["Records"], FacturaCompra);
	        this.TotalRecords = source["TotalRecords"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GmailAuthStatus {
	    authenticated: boolean;
	    credPresent: boolean;
	    configDir: string;
	
	    static createFrom(source: any = {}) {
	        return new GmailAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.credPresent = source["credPresent"];
	        this.configDir = source["configDir"];
	    }
	}
	export class GmailSyncProgress {
	    running: boolean;
	    fase: string;
	    total: number;
	    procesados: number;
	    nuevas: number;
	    duplicadas: number;
	    errores: number;
	    ultimoNro: string;
	
	    static createFrom(source: any = {}) {
	        return new GmailSyncProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.fase = source["fase"];
	        this.total = source["total"];
	        this.procesados = source["procesados"];
	        this.nuevas = source["nuevas"];
	        this.duplicadas = source["duplicadas"];
	        this.errores = source["errores"];
	        this.ultimoNro = source["ultimoNro"];
	    }
	}
	export class IndexInfo {
	    Name: string;
	    IsUnique: boolean;
	    IsPrimary: boolean;
	    Definition: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.IsUnique = source["IsUnique"];
	        this.IsPrimary = source["IsPrimary"];
	        this.Definition = source["Definition"];
	    }
	}
	export class LoginRequest {
	    Email: string;
	    Contrasena: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Email = source["Email"];
	        this.Contrasena = source["Contrasena"];
	    }
	}
	export class LoginResponse {
	    MFARequired: boolean;
	    Token: string;
	    Vendedor: Vendedor;
	
	    static createFrom(source: any = {}) {
	        return new LoginResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MFARequired = source["MFARequired"];
	        this.Token = source["Token"];
	        this.Vendedor = this.convertValues(source["Vendedor"], Vendedor);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MFASetupResponse {
	    Secret: string;
	    ImageURL: string;
	
	    static createFrom(source: any = {}) {
	        return new MFASetupResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Secret = source["Secret"];
	        this.ImageURL = source["ImageURL"];
	    }
	}
	
	export class NuevoProducto {
	    UUID: string;
	    VendedorUUID: string;
	    Nombre: string;
	    Codigo: string;
	    PrecioVenta: number;
	    Stock: number;
	
	    static createFrom(source: any = {}) {
	        return new NuevoProducto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.VendedorUUID = source["VendedorUUID"];
	        this.Nombre = source["Nombre"];
	        this.Codigo = source["Codigo"];
	        this.PrecioVenta = source["PrecioVenta"];
	        this.Stock = source["Stock"];
	    }
	}
	export class OperacionStock {
	    UUID: string;
	    ProductoUUID: string;
	    TipoOperacion: string;
	    CantidadCambio: number;
	    StockResultante: number;
	    VendedorUUID: string;
	    FacturaUUID?: string;
	    Timestamp: string;
	    Sincronizado: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OperacionStock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.ProductoUUID = source["ProductoUUID"];
	        this.TipoOperacion = source["TipoOperacion"];
	        this.CantidadCambio = source["CantidadCambio"];
	        this.StockResultante = source["StockResultante"];
	        this.VendedorUUID = source["VendedorUUID"];
	        this.FacturaUUID = source["FacturaUUID"];
	        this.Timestamp = source["Timestamp"];
	        this.Sincronizado = source["Sincronizado"];
	    }
	}
	export class OperationResult {
	    Success: boolean;
	    Message: string;
	
	    static createFrom(source: any = {}) {
	        return new OperationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Success = source["Success"];
	        this.Message = source["Message"];
	    }
	}
	export class PaginatedResult {
	    Records: any;
	    TotalRecords: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Records = source["Records"];
	        this.TotalRecords = source["TotalRecords"];
	    }
	}
	
	export class ProductoAjusteRequest {
	    UUID: string;
	    Nombre: string;
	    PrecioVenta: number;
	    Stock: number;
	    VendedorUUID?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProductoAjusteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.Nombre = source["Nombre"];
	        this.PrecioVenta = source["PrecioVenta"];
	        this.Stock = source["Stock"];
	        this.VendedorUUID = source["VendedorUUID"];
	    }
	}
	export class ProductoAlerta {
	    uuid: string;
	    nombre: string;
	    codigo: string;
	    stock: number;
	    precioVenta: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductoAlerta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.nombre = source["nombre"];
	        this.codigo = source["codigo"];
	        this.stock = source["stock"];
	        this.precioVenta = source["precioVenta"];
	    }
	}
	export class ProductoComprado {
	    Descripcion: string;
	    CodigoProducto: string;
	    TotalCantidad: number;
	    TotalComprado: number;
	    NumFacturas: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductoComprado(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Descripcion = source["Descripcion"];
	        this.CodigoProducto = source["CodigoProducto"];
	        this.TotalCantidad = source["TotalCantidad"];
	        this.TotalComprado = source["TotalComprado"];
	        this.NumFacturas = source["NumFacturas"];
	    }
	}
	
	export class ProductoVenta {
	    ProductoUUID: string;
	    Cantidad: number;
	    PrecioUnitario: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductoVenta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProductoUUID = source["ProductoUUID"];
	        this.Cantidad = source["Cantidad"];
	        this.PrecioUnitario = source["PrecioUnitario"];
	    }
	}
	export class Proveedor {
	    CreatedAt: string;
	    UpdatedAt: string;
	    DeletedAt?: string;
	    uuid: string;
	    NIT: string;
	    Nombre: string;
	    Telefono: string;
	    Email: string;
	
	    static createFrom(source: any = {}) {
	        return new Proveedor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	        this.DeletedAt = source["DeletedAt"];
	        this.uuid = source["uuid"];
	        this.NIT = source["NIT"];
	        this.Nombre = source["Nombre"];
	        this.Telefono = source["Telefono"];
	        this.Email = source["Email"];
	    }
	}
	export class ProveedorStats {
	    uuid: string;
	    NIT: string;
	    Nombre: string;
	    Telefono: string;
	    Email: string;
	    TotalFacturas: number;
	    TotalComprado: number;
	    UltimaCompra: string;
	    TopProductos: ProductoComprado[];
	
	    static createFrom(source: any = {}) {
	        return new ProveedorStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.NIT = source["NIT"];
	        this.Nombre = source["Nombre"];
	        this.Telefono = source["Telefono"];
	        this.Email = source["Email"];
	        this.TotalFacturas = source["TotalFacturas"];
	        this.TotalComprado = source["TotalComprado"];
	        this.UltimaCompra = source["UltimaCompra"];
	        this.TopProductos = this.convertValues(source["TopProductos"], ProductoComprado);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProveedoresStatsResponse {
	    Records: ProveedorStats[];
	    TotalRecords: number;
	
	    static createFrom(source: any = {}) {
	        return new ProveedoresStatsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Records = this.convertValues(source["Records"], ProveedorStats);
	        this.TotalRecords = source["TotalRecords"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReporteVentas {
	    totalVentas: number;
	    numeroVentas: number;
	    ticketPromedio: number;
	    ventasIndividuales: VentaIndividual[];
	    topProductos: ProductoVendido[];
	    topVendedores: VendedorRendimiento[];
	    metodosPago: any[];
	
	    static createFrom(source: any = {}) {
	        return new ReporteVentas(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalVentas = source["totalVentas"];
	        this.numeroVentas = source["numeroVentas"];
	        this.ticketPromedio = source["ticketPromedio"];
	        this.ventasIndividuales = this.convertValues(source["ventasIndividuales"], VentaIndividual);
	        this.topProductos = this.convertValues(source["topProductos"], ProductoVendido);
	        this.topVendedores = this.convertValues(source["topVendedores"], VendedorRendimiento);
	        this.metodosPago = source["metodosPago"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResumenCompras {
	    TotalGastado: number;
	    NumFacturas: number;
	    NumNotasCredito: number;
	    NumNotasDebito: number;
	    NumProveedores: number;
	    TopProveedores: ProveedorStats[];
	    TopProductos: ProductoComprado[];
	
	    static createFrom(source: any = {}) {
	        return new ResumenCompras(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TotalGastado = source["TotalGastado"];
	        this.NumFacturas = source["NumFacturas"];
	        this.NumNotasCredito = source["NumNotasCredito"];
	        this.NumNotasDebito = source["NumNotasDebito"];
	        this.NumProveedores = source["NumProveedores"];
	        this.TopProveedores = this.convertValues(source["TopProveedores"], ProveedorStats);
	        this.TopProductos = this.convertValues(source["TopProductos"], ProductoComprado);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResumenInventario {
	    totalProductos: number;
	    productosStockBajo: number;
	    productosSinStock: number;
	    valorInventario: number;
	    productosAlerta: ProductoAlerta[];
	
	    static createFrom(source: any = {}) {
	        return new ResumenInventario(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalProductos = source["totalProductos"];
	        this.productosStockBajo = source["productosStockBajo"];
	        this.productosSinStock = source["productosSinStock"];
	        this.valorInventario = source["valorInventario"];
	        this.productosAlerta = this.convertValues(source["productosAlerta"], ProductoAlerta);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SyncOptions {
	    modo: string;
	    desde: string;
	    hasta: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modo = source["modo"];
	        this.desde = source["desde"];
	        this.hasta = source["hasta"];
	    }
	}
	export class TableInfo {
	    Name: string;
	    Schema: string;
	    RowCount: number;
	    SizeBytes: number;
	    SizeHuman: string;
	
	    static createFrom(source: any = {}) {
	        return new TableInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Schema = source["Schema"];
	        this.RowCount = source["RowCount"];
	        this.SizeBytes = source["SizeBytes"];
	        this.SizeHuman = source["SizeHuman"];
	    }
	}
	export class TablePreview {
	    Columns: string[];
	    Rows: any[];
	    Total: number;
	    Limit: number;
	    Offset: number;
	
	    static createFrom(source: any = {}) {
	        return new TablePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Columns = source["Columns"];
	        this.Rows = source["Rows"];
	        this.Total = source["Total"];
	        this.Limit = source["Limit"];
	        this.Offset = source["Offset"];
	    }
	}
	export class TableSchema {
	    TableInfo: TableInfo;
	    Columns: ColumnInfo[];
	    Indexes: IndexInfo[];
	    Constraints: ConstraintInfo[];
	
	    static createFrom(source: any = {}) {
	        return new TableSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TableInfo = this.convertValues(source["TableInfo"], TableInfo);
	        this.Columns = this.convertValues(source["Columns"], ColumnInfo);
	        this.Indexes = this.convertValues(source["Indexes"], IndexInfo);
	        this.Constraints = this.convertValues(source["Constraints"], ConstraintInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TransferenciaBancolombia {
	    uuid: string;
	    emailId: string;
	    fecha: string;
	    monto: number;
	    remitente: string;
	    referencia: string;
	    cuentaDestino: string;
	    concepto: string;
	    rawSubject: string;
	    leido: boolean;
	    creadoEn: string;
	    facturaUuid: string;
	    facturaNumero: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferenciaBancolombia(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.emailId = source["emailId"];
	        this.fecha = source["fecha"];
	        this.monto = source["monto"];
	        this.remitente = source["remitente"];
	        this.referencia = source["referencia"];
	        this.cuentaDestino = source["cuentaDestino"];
	        this.concepto = source["concepto"];
	        this.rawSubject = source["rawSubject"];
	        this.leido = source["leido"];
	        this.creadoEn = source["creadoEn"];
	        this.facturaUuid = source["facturaUuid"];
	        this.facturaNumero = source["facturaNumero"];
	    }
	}
	export class TransferenciasResponse {
	    items: TransferenciaBancolombia[];
	    total: number;
	    page: number;
	    pageSize: number;
	    totalMonto: number;
	
	    static createFrom(source: any = {}) {
	        return new TransferenciasResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], TransferenciaBancolombia);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.totalMonto = source["totalMonto"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class VendedorUpdateRequest {
	    UUID: string;
	    Nombre: string;
	    Apellido: string;
	    Cedula: string;
	    Email: string;
	    ContrasenaActual?: string;
	    ContrasenaNueva?: string;
	
	    static createFrom(source: any = {}) {
	        return new VendedorUpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.Nombre = source["Nombre"];
	        this.Apellido = source["Apellido"];
	        this.Cedula = source["Cedula"];
	        this.Email = source["Email"];
	        this.ContrasenaActual = source["ContrasenaActual"];
	        this.ContrasenaNueva = source["ContrasenaNueva"];
	    }
	}
	
	export class VentaRequest {
	    ClienteUUID: string;
	    VendedorUUID: string;
	    Productos: ProductoVenta[];
	    MetodoPago: string;
	
	    static createFrom(source: any = {}) {
	        return new VentaRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ClienteUUID = source["ClienteUUID"];
	        this.VendedorUUID = source["VendedorUUID"];
	        this.Productos = this.convertValues(source["Productos"], ProductoVenta);
	        this.MetodoPago = source["MetodoPago"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace sql {
	
	export class DB {
	
	
	    static createFrom(source: any = {}) {
	        return new DB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Row {
	
	
	    static createFrom(source: any = {}) {
	        return new Row(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Rows {
	
	
	    static createFrom(source: any = {}) {
	        return new Rows(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Tx {
	
	
	    static createFrom(source: any = {}) {
	        return new Tx(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

