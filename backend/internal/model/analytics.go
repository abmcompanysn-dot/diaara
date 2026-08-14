package model

import "time"

// SalesByDayPoint est un point de la courbe de ventes/revenu par jour.
type SalesByDayPoint struct {
	Day        time.Time `json:"day"`
	Sales      int       `json:"sales"`
	RevenueCFA int       `json:"revenue_cfa"`
}

// TopProductPoint est une ligne du classement des produits par revenu.
type TopProductPoint struct {
	ProductID  string `json:"product_id"`
	Title      string `json:"title"`
	Sales      int    `json:"sales"`
	RevenueCFA int    `json:"revenue_cfa"`
}

// TopVendorPoint est une ligne du classement des vendeurs par revenu.
type TopVendorPoint struct {
	VendorID   string `json:"vendor_id"`
	Email      string `json:"email"`
	Sales      int    `json:"sales"`
	RevenueCFA int    `json:"revenue_cfa"`
}

// CloserConversionPoint est une ligne du classement des closers par conversion.
type CloserConversionPoint struct {
	CloserID      string  `json:"closer_id"`
	Email         string  `json:"email"`
	Clicks        int     `json:"clicks"`
	Sales         int     `json:"sales"`
	RevenueCFA    int     `json:"revenue_cfa"`
	ConversionPct float64 `json:"conversion_pct"`
}

// AnalyticsOverview combine les séries affichées sur la page analytique admin.
type AnalyticsOverview struct {
	SalesByDay  []SalesByDayPoint       `json:"sales_by_day"`
	TopProducts []TopProductPoint       `json:"top_products"`
	TopVendors  []TopVendorPoint        `json:"top_vendors"`
	TopClosers  []CloserConversionPoint `json:"top_closers"`
}

// SystemHealth reflète l'état des dépendances directes du process backend.
type SystemHealth struct {
	Database        string `json:"database"`         // "ok" | "error"
	Storage         string `json:"storage"`          // "ok" | "error" | "disabled"
	Email           string `json:"email"`            // "ok" | "disabled" (aucun fournisseur configuré)
	UptimeSeconds   int64  `json:"uptime_seconds"`
	GoroutineCount  int    `json:"goroutine_count"`
	MemAllocMB      float64 `json:"mem_alloc_mb"`
	MemSysMB        float64 `json:"mem_sys_mb"`
}
