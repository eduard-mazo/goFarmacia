package backend

import (
	"testing"
)

func TestParseBancolombiaTransfer(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantMonto  float64
		wantRemit  string
		wantCuenta string
	}{
		{
			name: "Formato A — $65.000 sin decimales",
			text: "Bancolombia: Recibiste una transferencia por $65,000 de DAVID SIERRA en tu cuenta **8368, el 07/05/2025 a las 13:59. Si tienes dudas, hablemos: 018000931987. Siempre a tu lado.",
			wantMonto:  65000,
			wantRemit:  "DAVID SIERRA",
			wantCuenta: "*8368",
		},
		{
			name: "Formato A — $3.500 remitente corto",
			text: "Bancolombia: Recibiste una transferencia por $3,500 de LUIS MAYA en tu cuenta **8368, el 26/01/2025 a las 21:13. Si tienes dudas, hablemos: 018000931987. Siempre a tu lado.",
			wantMonto:  3500,
			wantRemit:  "LUIS MAYA",
			wantCuenta: "*8368",
		},
		{
			name: "Formato B — llave, fecha corta DD/MM/YY",
			text: "Bancolombia: EDUARD, recibiste una transferencia de SANDRA LORENA VALENCIA CORREA por $70,000.00 en tu producto *8368 conectado a la llave eduard.mazo@gmail.com el 27/05/25 a las 13:05. Con llaves es de una y gratis. Dudas al 018000931987.",
			wantMonto:  70000,
			wantRemit:  "SANDRA LORENA VALENCIA CORREA",
			wantCuenta: "*8368",
		},
		{
			name: "Formato B — $10.000 HTML stripped",
			text: "Bancolombia: EDUARD, recibiste una transferencia de MARCELA JIMENEZ SANCHEZ por $10,000.00 en tu cuenta *8368 conectada a la llave eduard.mazo@gmail.com el 06/02/26 a las 12:37. Con llaves es de una y gratis. Dudas al 018000912345.",
			wantMonto:  10000,
			wantRemit:  "MARCELA JIMENEZ SANCHEZ",
			wantCuenta: "*8368",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBancolombiaTransfer(tc.text, "test-email-id", "")
			if got == nil {
				t.Fatalf("parseBancolombiaTransfer() returned nil, want non-nil")
			}
			if got.Monto != tc.wantMonto {
				t.Errorf("Monto = %.2f, want %.2f", got.Monto, tc.wantMonto)
			}
			if got.Remitente != tc.wantRemit {
				t.Errorf("Remitente = %q, want %q", got.Remitente, tc.wantRemit)
			}
			if got.CuentaDestino != tc.wantCuenta {
				t.Errorf("CuentaDestino = %q, want %q", got.CuentaDestino, tc.wantCuenta)
			}
		})
	}
}

func TestParseBancolombiaAmount(t *testing.T) {
	cases := []struct{ in string; want float64 }{
		{"65,000", 65000},
		{"3,500", 3500},
		{"70,000.00", 70000},
		{"10,000.00", 10000},
		{"1,500,000", 1500000},
		{"500", 500},
	}
	for _, c := range cases {
		got := parseBancolombiaAmount(c.in)
		if got != c.want {
			t.Errorf("parseBancolombiaAmount(%q) = %.2f, want %.2f", c.in, got, c.want)
		}
	}
}
