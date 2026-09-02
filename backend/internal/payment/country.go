package payment

import (
	"sort"
	"strings"
)

// CountryName — nom lisible (français) d'un pays couvert, indexé par code
// ISO 3166-1 alpha-3. Couvre les 20 pays PawaPay (mêmes que XOFOperators /
// CountryCurrency) plus rien d'autre : un code absent est un pays hors zone.
var CountryName = map[string]string{
	"SEN": "Sénégal", "CIV": "Côte d'Ivoire", "BEN": "Bénin", "BFA": "Burkina Faso",
	"CMR": "Cameroun", "GAB": "Gabon", "COG": "Congo-Brazzaville", "COD": "RD Congo",
	"GHA": "Ghana", "NGA": "Nigeria", "KEN": "Kenya", "RWA": "Rwanda", "UGA": "Ouganda",
	"TZA": "Tanzanie", "ZMB": "Zambie", "MWI": "Malawi", "MOZ": "Mozambique",
	"LSO": "Lesotho", "SLE": "Sierra Leone", "ETH": "Éthiopie",
}

// dialCodeToCountry — indicatif téléphonique international → code ISO3. Dérivé
// une seule fois de XOFOperators (source unique des indicatifs). Un indicatif
// partagé par plusieurs pays de la liste n'existe pas ici (chaque DialCode de
// XOFOperators est propre à un pays).
var dialCodeToCountry = func() map[string]string {
	m := make(map[string]string, len(XOFOperators))
	for _, o := range XOFOperators {
		m[o.DialCode] = o.Country
	}
	return m
}()

// dialCodesByLengthDesc — tous les indicatifs connus, triés du plus long au
// plus court, pour tester les préfixes sans ambiguïté (ex. "255" avant "25").
var dialCodesByLengthDesc = func() []string {
	codes := make([]string, 0, len(dialCodeToCountry))
	for c := range dialCodeToCountry {
		codes = append(codes, c)
	}
	sort.Slice(codes, func(i, j int) bool {
		if len(codes[i]) != len(codes[j]) {
			return len(codes[i]) > len(codes[j])
		}
		return codes[i] < codes[j]
	})
	return codes
}()

// CountryFromPhone déduit le code pays ISO3 d'un numéro de téléphone au format
// international (avec ou sans "+", espaces tolérés). Renvoie "" si le numéro
// est vide, non international, ou d'un pays hors des 20 pays couverts.
func CountryFromPhone(phone string) string {
	digits := nonDigits.ReplaceAllString(phone, "")
	if digits == "" {
		return ""
	}
	for _, code := range dialCodesByLengthDesc {
		if strings.HasPrefix(digits, code) {
			return dialCodeToCountry[code]
		}
	}
	return ""
}

// CountryLabel renvoie le nom lisible d'un code ISO3, ou le code lui-même s'il
// n'est pas dans la liste (et "—" pour une chaîne vide).
func CountryLabel(iso3 string) string {
	if iso3 == "" {
		return "—"
	}
	if name, ok := CountryName[strings.ToUpper(iso3)]; ok {
		return name
	}
	return iso3
}
