export namespace backend {
	
	export class Cliente {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
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
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.Nombre = source["Nombre"];
	        this.Apellido = source["Apellido"];
	        this.TipoID = source["TipoID"];
	        this.NumeroID = source["NumeroID"];
	        this.Telefono = source["Telefono"];
	        this.Email = source["Email"];
	        this.Direccion = source["Direccion"];
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
	export class Producto {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    Nombre: string;
	    Codigo: string;
	    PrecioVenta: number;
	    Stock: number;
	
	    static createFrom(source: any = {}) {
	        return new Producto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.Nombre = source["Nombre"];
	        this.Codigo = source["Codigo"];
	        this.PrecioVenta = source["PrecioVenta"];
	        this.Stock = source["Stock"];
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
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    FacturaID: number;
	    ProductoID: number;
	    Producto: Producto;
	    Cantidad: number;
	    PrecioUnitario: number;
	    PrecioTotal: number;
	
	    static createFrom(source: any = {}) {
	        return new DetalleFactura(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.FacturaID = source["FacturaID"];
	        this.ProductoID = source["ProductoID"];
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
	export class Vendedor {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    Nombre: string;
	    Apellido: string;
	    Cedula: string;
	    Email: string;
	    Contrasena: string;
	
	    static createFrom(source: any = {}) {
	        return new Vendedor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.Nombre = source["Nombre"];
	        this.Apellido = source["Apellido"];
	        this.Cedula = source["Cedula"];
	        this.Email = source["Email"];
	        this.Contrasena = source["Contrasena"];
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
	export class Factura {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    NumeroFactura: string;
	    // Go type: time
	    fecha_emision: any;
	    VendedorID: number;
	    Vendedor: Vendedor;
	    ClienteID: number;
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
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.NumeroFactura = source["NumeroFactura"];
	        this.fecha_emision = this.convertValues(source["fecha_emision"], null);
	        this.VendedorID = source["VendedorID"];
	        this.Vendedor = this.convertValues(source["Vendedor"], Vendedor);
	        this.ClienteID = source["ClienteID"];
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
	
	export class ProductoVenta {
	    ID: number;
	    Cantidad: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductoVenta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Cantidad = source["Cantidad"];
	    }
	}
	
	export class VentaRequest {
	    ClienteID: number;
	    VendedorID: number;
	    Productos: ProductoVenta[];
	    MetodoPago: string;
	
	    static createFrom(source: any = {}) {
	        return new VentaRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ClienteID = source["ClienteID"];
	        this.VendedorID = source["VendedorID"];
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

export namespace gorm {
	
	export class DeletedAt {
	    // Go type: time
	    Time: any;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeletedAt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], null);
	        this.Valid = source["Valid"];
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

