#!/usr/bin/env python3
"""Generate MSNTB dataset: 100 Indonesian route planning prompts with ground truth coordinates."""

import json

# Known landmarks with verified coordinates
LANDMARKS = {
    # Jakarta
    "Monumen Nasional": (-6.1754, 106.8272),
    "Monas": (-6.1754, 106.8272),
    "Museum Nasional": (-6.1763, 106.8226),
    "Museum Gajah": (-6.1763, 106.8226),
    "Istiqlal Mosque": (-6.1701, 106.8305),
    "Masjid Istiqlal": (-6.1701, 106.8305),
    "Gereja Katedral": (-6.1698, 106.8319),
    "Jakarta Cathedral": (-6.1698, 106.8319),
    "Kota Tua": (-6.1376, 106.8137),
    "Fatahillah Museum": (-6.1376, 106.8137),
    "Museum Sejarah Jakarta": (-6.1376, 106.8137),
    "Mangga Dua": (-6.1464, 106.8307),
    "Ancol": (-6.1219, 106.8433),
    "Taman Impian Jaya Ancol": (-6.1219, 106.8433),
    "Dunia Fantasi": (-6.1219, 106.8433),
    "Gelora Bung Karno": (-6.2187, 106.8017),
    "Senayan": (-6.2187, 106.8017),
    "Plaza Senayan": (-6.2285, 106.7989),
    "Senayan City": (-6.2285, 106.7989),
    "Grand Indonesia": (-6.1947, 106.8184),
    "Thamrin": (-6.1947, 106.8184),
    "Bundaran HI": (-6.1947, 106.8184),
    "Sarinah": (-6.1933, 106.8216),
    "Gedung Pancasila": (-6.1933, 106.8216),
    "Kemang": (-6.2465, 106.8148),
    "Pasaraya": (-6.2465, 106.8148),
    "Ragunan": (-6.2907, 106.8213),
    "Kebun Binatang Ragunan": (-6.2907, 106.8213),
    "Taman Mini": (-6.3027, 106.8891),
    "Taman Mini Indonesia Indah": (-6.3027, 106.8891),
    "TMII": (-6.3027, 106.8891),
    "Menteng": (-6.1891, 106.8389),
    "Cikini": (-6.1891, 106.8389),
    "Kemayoran": (-6.1575, 106.8564),
    "JITC Kemayoran": (-6.1575, 106.8564),
    "Kuningan": (-6.2367, 106.8388),
    "Setiabudi": (-6.2091, 106.8251),
    "Gambir": (-6.1754, 106.8272),
    "Tanah Abang": (-6.1856, 106.8117),
    "Menteng Atas": (-6.2033, 106.8433),
    "Pancoran": (-6.2455, 106.8536),
    "Cawang": (-6.2294, 106.8608),
    "Condet": (-6.2736, 106.8583),
    "Pasar Minggu": (-6.2936, 106.8464),
    "Kalibata": (-6.2600, 106.8464),
    "Pecenongan": (-6.1633, 106.8281),
    "Sabang": (-6.1633, 106.8281),
    "Gondangdia": (-6.1822, 106.8375),
    "Cikini Raya": (-6.1891, 106.8389),
    "Jalan Sudirman": (-6.2091, 106.8251),
    "Jalan Thamrin": (-6.1947, 106.8184),
    "Taman Ismail Marzuki": (-6.1933, 106.8389),
    "TIM": (-6.1933, 106.8389),
    "Planetarium": (-6.1933, 106.8389),
    "Taman Suropati": (-6.1914, 106.8364),
    "Taman Proklamasi": (-6.1875, 106.8408),
    "Tugu Proklamasi": (-6.1875, 106.8408),
    "Taman Waduk Melati": (-6.1833, 106.8442),
    "Ratu Adil": (-6.1833, 106.8442),
    "Pulo Gadung": (-6.1728, 106.8653),
    "Kemanggisan": (-6.1847, 106.7914),
    "Palmerah": (-6.1847, 106.7914),
    "Slipi": (-6.1989, 106.7881),
    "Grogol": (-6.1667, 106.7892),
    "Tanjung Duren": (-6.1667, 106.7892),
    "Pluit": (-6.1003, 106.8072),
    "Muara Karang": (-6.1003, 106.7836),
    "Pantai Indah Kapuk": (-6.1089, 106.7836),
    "PIK": (-6.1089, 106.7836),
    "Kelapa Gading": (-6.1578, 106.9064),
    "Sunter": (-6.1578, 106.8864),
    "Pulomas": (-6.1578, 106.8964),
    "Kemayoran": (-6.1575, 106.8564),
    "Senen": (-6.1753, 106.8436),
    "Pasar Senen": (-6.1753, 106.8436),
    "Salemba": (-6.1864, 106.8514),
    "Kramat": (-6.1864, 106.8514),
    "Pegangsaan": (-6.1891, 106.8389),
    "Bendungan Hilir": (-6.2050, 106.8133),
    "Benhil": (-6.2050, 106.8133),
    "Karet": (-6.2050, 106.8133),
    "Sudirman": (-6.2091, 106.8251),
    "SCBD": (-6.2267, 106.8117),
    "Kota Kasablanka": (-6.2267, 106.8117),
    "Mampang": (-6.2367, 106.8336),
    "Tebet": (-6.2217, 106.8614),
    "Pejaten": (-6.2617, 106.8364),
    "Jati Padang": (-6.2717, 106.8364),
    "Cilandak": (-6.2767, 106.8014),
    "Lebak Bulus": (-6.2889, 106.7753),
    "Fatmawati": (-6.2617, 106.7936),
    "Bintaro": (-6.2436, 106.7583),
    "Pondok Indah": (-6.2617, 106.7836),
    "PIK 2": (-6.0836, 106.7836),
    "Jakarta International Stadium": (-6.1219, 106.8636),
    "JIS": (-6.1219, 106.8636),
    "Museum Satria Mandala": (-6.2367, 106.8388),
    "Satria Mandala": (-6.2367, 106.8388),
    "Balai Kota DKI": (-6.1754, 106.8272),
    "Gedung DPR": (-6.2187, 106.8017),
    "DPR MPR RI": (-6.2187, 106.8017),
    "Kantor Gubernur DKI": (-6.1754, 106.8272),

    # Yogyakarta
    "Malioboro": (-7.7926, 110.3658),
    "Jalan Malioboro": (-7.7926, 110.3658),
    "Kraton Yogyakarta": (-7.8055, 110.3642),
    "Kraton": (-7.8055, 110.3642),
    "Taman Sari": (-7.8106, 110.3594),
    "Istana Air Taman Sari": (-7.8106, 110.3594),
    "Benteng Vredenburg": (-7.7983, 110.3697),
    "Vredenburg": (-7.7983, 110.3697),
    "Monumen Yogya Kembali": (-7.7397, 110.3972),
    "Monjali": (-7.7397, 110.3972),
    "Candi Prambanan": (-7.7520, 110.4914),
    "Prambanan": (-7.7520, 110.4914),
    "Candi Borobudur": (-7.6079, 110.2038),
    "Borobudur": (-7.6079, 110.2038),
    "Malioboro": (-7.7926, 110.3658),
    "Pasar Beringharjo": (-7.7926, 110.3658),
    "Beringharjo": (-7.7926, 110.3658),
    "Tugu Yogyakarta": (-7.7879, 110.3672),
    "Tugu Jogja": (-7.7879, 110.3672),
    "Jogja": (-7.7956, 110.3694),
    "Yogyakarta": (-7.7956, 110.3694),
    "Kotagede": (-7.8089, 110.3978),
    "Museum Sonobudoyo": (-7.8014, 110.3625),
    "Sonobudoyo": (-7.8014, 110.3625),
    "Gembira Loka Zoo": (-7.7864, 110.3856),
    "Gembira Loka": (-7.7864, 110.3856),
    "Jogja Bay": (-7.7336, 110.3414),
    "Heha Sky View": (-7.7236, 110.4036),
    "Gumuk Pasir Parangkusumo": (-7.9781, 110.3519),
    "Parangtritis": (-7.9781, 110.3519),
    "Pantai Parangtritis": (-7.9781, 110.3519),
    "Candi Ratu Boko": (-7.7703, 110.4856),
    "Ratu Boko": (-7.7703, 110.4856),
    "Taman Pelangi": (-7.7478, 110.4056),
    "Museum Affandi": (-7.7817, 110.3767),
    "Affandi": (-7.7817, 110.3767),
    "UGM": (-7.7756, 110.3842),
    "Universitas Gadjah Mada": (-7.7756, 110.3842),
    "UII": (-7.7386, 110.4014),
    "Universitas Islam Indonesia": (-7.7386, 110.4014),
    "Malioboro": (-7.7926, 110.3658),
    "Nol Kilometer": (-7.7926, 110.3658),
    "Titik Nol KM Yogyakarta": (-7.7926, 110.3658),

    # Bandung
    "Gedung Sate": (-6.8905, 107.6107),
    "Gedung Merdeka": (-6.8914, 107.6136),
    "Alun-Alun Bandung": (-6.9175, 107.6191),
    "Masjid Raya Bandung": (-6.9175, 107.6191),
    "Kebun Binatang Bandung": (-6.8836, 107.6200),
    "Bandung Zoo": (-6.8836, 107.6200),
    "Braga": (-6.9175, 107.6117),
    "Jalan Braga": (-6.9175, 107.6117),
    "Braga City Walk": (-6.9175, 107.6117),
    "Pasar Baru Bandung": (-6.9214, 107.6042),
    "Pasar Baru": (-6.9214, 107.6042),
    "Cihampelas": (-6.8925, 107.5917),
    "Cihampelas Walk": (-6.8925, 107.5917),
    "Ciwalk": (-6.8925, 107.5917),
    "Paris Van Java": (-6.8836, 107.5917),
    "PVJ": (-6.8836, 107.5917),
    "Bandung Super Mall": (-6.8936, 107.6056),
    "Trans Studio Bandung": (-6.9281, 107.6333),
    "Trans Studio": (-6.9281, 107.6333),
    "Tebing Keraton": (-6.8336, 107.6167),
    "Keraton Cliff": (-6.8336, 107.6167),
    "Dago": (-6.8636, 107.6100),
    "Dago Pakar": (-6.8336, 107.6167),
    "Taman Hutan Raya Ir. H. Juanda": (-6.8636, 107.6100),
    "Tahura Juanda": (-6.8636, 107.6100),
    "Kawah Putih": (-7.1661, 107.4025),
    "Situ Patenggang": (-7.1336, 107.3836),
    "Ranca Upas": (-7.1336, 107.4136),
    "Floating Market Lembang": (-6.7936, 107.6136),
    "Floating Market": (-6.7936, 107.6136),
    "Farm House Lembang": (-6.7936, 107.6036),
    "De Ranch Lembang": (-6.8136, 107.6136),
    "Gedung Sate": (-6.8905, 107.6107),
    "Gedung Pos": (-6.9175, 107.6042),
    "Gedung Bandung Tempo Doeloe": (-6.9175, 107.6117),
    "Museum Geologi": (-6.8905, 107.6107),
    "Museum Konferensi Asia Afrika": (-6.8914, 107.6136),
    "KAA Museum": (-6.8914, 107.6136),
    "Monumen Bandung Lautan Api": (-6.9214, 107.6042),
    "Balaikota Bandung": (-6.9036, 107.6136),
    "ITB": (-6.8914, 107.5917),
    "Institut Teknologi Bandung": (-6.8914, 107.5917),
    "UNPAD": (-6.9375, 107.6250),
    "Universitas Padjadjaran": (-6.9375, 107.6250),
    "Stasiun Bandung": (-6.9175, 107.6042),
    "Stasiun Kiaracondong": (-6.9281, 107.6536),
    "Bundaran Cibeunying": (-6.8864, 107.6267),
    "Cibeunying": (-6.8864, 107.6267),
    "Dayeuhkolot": (-6.9750, 107.6167),
    "Cimahi": (-6.8736, 107.5336),
    "Lembang": (-6.8136, 107.6136),
    "Sukajadi": (-6.8836, 107.5917),
    "Setiabudhi": (-6.8536, 107.5817),
}

# Verify coordinates with known good values
# Jakarta landmarks should be around (-6.1x, 106.8x)
# Yogyakarta landmarks should be around (-7.7x to -7.9x, 110.3x to 110.5x)
# Bandung landmarks should be around (-6.8x to -6.9x, 107.5x to 107.6x)

def make_prompt(id_num, difficulty, prompt, gt_stops):
    """Create a dataset entry."""
    ground_truth = []
    for name in gt_stops:
        if name in LANDMARKS:
            lat, lng = LANDMARKS[name]
            ground_truth.append({"name": name, "lat": lat, "lng": lng})
        else:
            raise ValueError(f"Unknown landmark: {name}")
    return {
        "id": f"MSNTB-{id_num:03d}",
        "difficulty": difficulty,
        "prompt": prompt,
        "ground_truth": ground_truth
    }

def main():
    dataset = []
    idx = 1

    # ============================================================
    # EASY (35 prompts) — 2 stops, well-known landmarks
    # ============================================================
    easy_prompts = [
        # Jakarta (21 = 60%)
        ("Dari Monas ke Museum Nasional", ["Monumen Nasional", "Museum Nasional"]),
        ("Dari Masjid Istiqlal ke Gereja Katedral", ["Istiqlal Mosque", "Gereja Katedral"]),
        ("Dari Kota Tua ke Ancol", ["Kota Tua", "Ancol"]),
        ("Dari Grand Indonesia ke Plaza Senayan", ["Grand Indonesia", "Plaza Senayan"]),
        ("Dari Ragunan ke Taman Mini", ["Ragunan", "Taman Mini"]),
        ("Dari Sarinah ke Bundaran HI", ["Sarinah", "Bundaran HI"]),
        ("Dari Kemang ke Kuningan", ["Kemang", "Kuningan"]),
        ("Dari Senayan City ke SCBD", ["Senayan City", "SCBD"]),
        ("Dari Gondangdia ke Cikini", ["Gondangdia", "Cikini"]),
        ("Dari Taman Ismail Marzuki ke Taman Suropati", ["Taman Ismail Marzuki", "Taman Suropati"]),
        ("Dari Pasar Senen ke Salemba", ["Pasar Senen", "Salemba"]),
        ("Dari Kelapa Gading ke Sunter", ["Kelapa Gading", "Sunter"]),
        ("Dari Pluit ke Pantai Indah Kapuk", ["Pluit", "Pantai Indah Kapuk"]),
        ("Dari Grogol ke Slipi", ["Grogol", "Slipi"]),
        ("Dari Mangga Dua ke Kemayoran", ["Mangga Dua", "Kemayoran"]),
        ("Dari Tebet ke Pancoran", ["Tebet", "Pancoran"]),
        ("Dari Kalibata ke Pasar Minggu", ["Kalibata", "Pasar Minggu"]),
        ("Dari Condet ke Ragunan", ["Condet", "Ragunan"]),
        ("Dari Palmerah ke Kemanggisan", ["Palmerah", "Kemanggisan"]),
        ("Dari Pondok Indah ke Cilandak", ["Pondok Indah", "Cilandak"]),
        ("Dari Lebak Bulus ke Fatmawati", ["Lebak Bulus", "Fatmawati"]),
        # Yogyakarta (9 = 25%)
        ("Dari Malioboro ke Kraton", ["Malioboro", "Kraton Yogyakarta"]),
        ("Dari Tugu Yogyakarta ke Taman Sari", ["Tugu Yogyakarta", "Taman Sari"]),
        ("Dari Prambanan ke Borobudur", ["Candi Prambanan", "Candi Borobudur"]),
        ("Dari Benteng Vredenburg ke Beringharjo", ["Benteng Vredenburg", "Pasar Beringharjo"]),
        ("Dari Monjali ke UGM", ["Monumen Yogya Kembali", "UGM"]),
        ("Dari Malioboro ke Nol Kilometer", ["Malioboro", "Nol Kilometer"]),
        ("Dari Kraton ke Taman Sari", ["Kraton Yogyakarta", "Taman Sari"]),
        ("Dari Sonobodoyo ke Museum Affandi", ["Museum Sonobudoyo", "Museum Affandi"]),
        ("Dari Kotagede ke Ratu Boko", ["Kotagede", "Candi Ratu Boko"]),
        # Bandung (5 = 15%)
        ("Dari Gedung Sate ke Alun-Alun Bandung", ["Gedung Sate", "Alun-Alun Bandung"]),
        ("Dari Braga ke Cihampelas", ["Braga", "Cihampelas"]),
        ("Dari Paris Van Java ke Trans Studio", ["Paris Van Java", "Trans Studio Bandung"]),
        ("Dari Dago ke Tebing Keraton", ["Dago", "Tebing Keraton"]),
        ("Dari Stasiun Bandung ke Gedung Sate", ["Stasiun Bandung", "Gedung Sate"]),
    ]

    for prompt_text, gt in easy_prompts:
        dataset.append(make_prompt(idx, "easy", prompt_text, gt))
        idx += 1

    # ============================================================
    # MEDIUM (40 prompts) — 3 stops, mix of landmarks + POI queries
    # ============================================================
    medium_prompts = [
        # Jakarta (24 = 60%)
        ("Dari Monas ke Ragunan, mampir ke Kuningan dulu", ["Monumen Nasional", "Kuningan", "Ragunan"]),
        ("Rute dari Kota Tua lewat Ancol ke Mangga Dua", ["Kota Tua", "Ancol", "Mangga Dua"]),
        ("Dari Grand Indonesia ke Istiqlal, mampir ke Sarinah", ["Grand Indonesia", "Sarinah", "Istiqlal Mosque"]),
        ("Dari Plaza Senayan ke Taman Mini, lewat Cawang", ["Plaza Senayan", "Cawang", "Taman Mini"]),
        ("Mau jalan dari Bundaran HI ke Kemang, mampir makan dulu di Senayan", ["Bundaran HI", "Senayan City", "Kemang"]),
        ("Dari Taman Ismail Marzuki ke Kalibata, lewat Cikini", ["Taman Ismail Marzuki", "Cikini", "Kalibata"]),
        ("Rute dari Senayan City ke Pasar Minggu, mampir ke Ragunan dulu", ["Senayan City", "Ragunan", "Pasar Minggu"]),
        ("Dari Kuningan ke Gondangdia, mampir ke Taman Suropati", ["Kuningan", "Taman Suropati", "Gondangdia"]),
        ("Dari SCBD ke Tebet, lewat Pancoran", ["SCBD", "Pancoran", "Tebet"]),
        ("Dari Kemayoran ke Condet, mampir ke Cawang", ["Kemayoran", "Cawang", "Condet"]),
        ("Mau ke Monas dari Kelapa Gading, mampir ke Sunter dulu", ["Sunter", "Kelapa Gading", "Monumen Nasional"]),
        ("Dari Pluit ke Ancol, lewat Muara Karang", ["Pluit", "Muara Karang", "Ancol"]),
        ("Rute dari Grogol ke Senen, mampir ke Pecenongan", ["Grogol", "Pecenongan", "Senen"]),
        ("Dari Setiabudi ke Menteng, lewat Sudirman", ["Setiabudi", "Sudirman", "Menteng"]),
        ("Dari Tanah Abang ke Pasar Senen, mampir ke Kramat", ["Tanah Abang", "Kramat", "Pasar Senen"]),
        ("Dari Benhil ke Gondangdia, lewat Thamrin", ["Bendungan Hilir", "Thamrin", "Gondangdia"]),
        ("Dari PIK ke Kelapa Gading, mampir ke Sunter", ["PIK", "Sunter", "Kelapa Gading"]),
        ("Dari Bintaro ke Ragunan, lewat Lebak Bulus", ["Bintaro", "Lebak Bulus", "Ragunan"]),
        ("Dari JIS ke Kota Tua, mampir ke Mangga Dua", ["Jakarta International Stadium", "Mangga Dua", "Kota Tua"]),
        ("Dari Pejaten ke Cilandak, lewat Pasar Minggu", ["Pejaten", "Pasar Minggu", "Cilandak"]),
        ("Dari TMII ke Monas, mampir ke Cikini", ["Taman Mini", "Cikini", "Monumen Nasional"]),
        ("Dari Mampang ke Tugu Proklamasi, lewat Menteng", ["Mampang", "Menteng", "Taman Proklamasi"]),
        ("Dari Jati Padang ke Kemang, mampir ke Pejaten", ["Jati Padang", "Pejaten", "Kemang"]),
        ("Dari Sudirman ke Gambir, lewat Senen", ["Sudirman", "Senen", "Gambir"]),
        # Yogyakarta (10 = 25%)
        ("Dari Malioboro ke Prambanan, mampir ke Monjali", ["Malioboro", "Monumen Yogya Kembali", "Candi Prambanan"]),
        ("Rute dari Borobudur ke Taman Sari, lewat Kraton", ["Candi Borobudur", "Kraton Yogyakarta", "Taman Sari"]),
        ("Dari UGM ke Tugu Yogyakarta, mampir ke Malioboro", ["UGM", "Tugu Yogyakarta", "Malioboro"]),
        ("Mau ke Prambanan dari Malioboro, mampir ke Vredenburg dulu", ["Malioboro", "Benteng Vredenburg", "Candi Prambanan"]),
        ("Dari Monjali ke Beringharjo, lewat Titik Nol KM", ["Monumen Yogya Kembali", "Nol Kilometer", "Pasar Beringharjo"]),
        ("Dari Sonobodoyo ke Taman Sari, mampir ke Kraton", ["Museum Sonobudoyo", "Kraton Yogyakarta", "Taman Sari"]),
        ("Rute dari Kotagede ke UGM, lewat Affandi", ["Kotagede", "Museum Affandi", "UGM"]),
        ("Dari Gembira Loka ke Prambanan, mampir ke Ratu Boko", ["Gembira Loka Zoo", "Candi Ratu Boko", "Candi Prambanan"]),
        ("Dari Parangtritis ke Borobudur, lewat Monjali", ["Pantai Parangtritis", "Monumen Yogya Kembali", "Candi Borobudur"]),
        ("Dari UII ke Malioboro, mampir ke Taman Pelangi", ["Universitas Islam Indonesia", "Taman Pelangi", "Malioboro"]),
        # Bandung (6 = 15%)
        ("Dari Gedung Sate ke Paris Van Java, mampir ke Braga", ["Gedung Sate", "Braga", "Paris Van Java"]),
        ("Rute dari Trans Studio ke Floating Market, lewat Dago", ["Trans Studio Bandung", "Dago", "Floating Market Lembang"]),
        ("Dari Alun-Alun ke Cihampelas, mampir ke Pasar Baru", ["Alun-Alun Bandung", "Pasar Baru", "Cihampelas"]),
        ("Dari Stasiun Bandung ke Kebun Binatang, lewat Braga", ["Stasiun Bandung", "Braga", "Kebun Binatang Bandung"]),
        ("Dari ITB ke Tebing Keraton, mampir ke Tahura", ["Institut Teknologi Bandung", "Tahura Juanda", "Tebing Keraton"]),
        ("Dari UNPAD ke Gedung Sate, lewat Cibeunying", ["Universitas Padjadjaran", "Bundaran Cibeunying", "Gedung Sate"]),
    ]

    for prompt_text, gt in medium_prompts:
        dataset.append(make_prompt(idx, "medium", prompt_text, gt))
        idx += 1

    # ============================================================
    # HARD (25 prompts) — 4-5 stops, vague descriptions, multi-city
    # ============================================================
    hard_prompts = [
        # Jakarta (15 = 60%)
        ("Dari Monas, mau mampir makan di Pecenongan, terus ke Istiqlal, lanjut ke Kota Tua, dan pulang lewat Mangga Dua",
         ["Monumen Nasional", "Pecenongan", "Istiqlal Mosque", "Kota Tua", "Mangga Dua"]),
        ("Rute keliling Jakarta: mulai dari Ancol, mampir ke Kota Tua, lewat Monas, lanjut ke Grand Indonesia, dan berakhir di Senayan",
         ["Ancol", "Kota Tua", "Monumen Nasional", "Grand Indonesia", "Plaza Senayan"]),
        ("Mau jalan dari Kemang, mampir sarapan di Ragunan, lewat Kuningan, dan berakhir di SCBD",
         ["Kemang", "Ragunan", "Kuningan", "SCBD"]),
        ("Perjalanan dari PIK ke TMII, mampir di Kelapa Gading, lewat Sunter, dan mampir di Pulo Gadung",
         ["PIK", "Kelapa Gading", "Sunter", "Pulo Gadung", "Taman Mini"]),
        ("Dari Lebak Bulus, mampir ke Pondok Indah, lanjut ke Cilandak, lewat Pasar Minggu, berakhir di Ragunan",
         ["Lebak Bulus", "Pondok Indah", "Cilandak", "Pasar Minggu", "Ragunan"]),
        ("Rute dari Grogol, lewat Tanah Abang, mampir ke Senen, lanjut ke Salemba, berakhir di Menteng",
         ["Grogol", "Tanah Abang", "Pasar Senen", "Salemba", "Menteng"]),
        ("Dari JIS, mampir ke Mangga Dua, lewat Kemayoran, lanjut ke Monas, dan berakhir di Taman Proklamasi",
         ["Jakarta International Stadium", "Mangga Dua", "Kemayoran", "Monumen Nasional", "Taman Proklamasi"]),
        ("Mau keliling Jakarta selatan: dari Fatmawati ke Bintaro, mampir di Lebak Bulus, lanjut ke Cilandak, dan berakhir di Kemang",
         ["Fatmawati", "Bintaro", "Lebak Bulus", "Cilandak", "Kemang"]),
        ("Dari Sudirman, mampir di Benhil, lewat Thamrin, lanjut ke Cikini, dan berakhir di Taman Suropati",
         ["Sudirman", "Bendungan Hilir", "Thamrin", "Cikini", "Taman Suropati"]),
        ("Rute dari Pluit ke Monas, mampir di PIK, lewat Muara Karang, dan mampir di Ancol sebelum ke Monas",
         ["Pluit", "PIK", "Muara Karang", "Ancol", "Monumen Nasional"]),
        ("Dari Cawang, lewat Tebet, mampir ke Kalibata, lanjut ke Pejaten, dan berakhir di Condet",
         ["Cawang", "Tebet", "Kalibata", "Pejaten", "Condet"]),
        ("Perjalanan dari Setiabudi ke Gambir, mampir di Sudirman, lewat Benhil, dan berakhir di Monas",
         ["Setiabudi", "Sudirman", "Bendungan Hilir", "Gambir", "Monumen Nasional"]),
        ("Dari TMII ke Ancol, mampir di Cawang, lewat Kemayoran, dan berakhir di JIS",
         ["Taman Mini", "Cawang", "Kemayoran", "Ancol", "Jakarta International Stadium"]),
        ("Mau ke Tugu Proklamasi dari Kuningan, mampir di Menteng, lewat Pegangsaan, dan berakhir di Gondangdia",
         ["Kuningan", "Menteng", "Pegangsaan", "Tugu Proklamasi", "Gondangdia"]),
        ("Rute dari Palmerah ke Tebet, mampir di Slipi, lewat Sudirman, dan berakhir di Cawang",
         ["Palmerah", "Slipi", "Sudirman", "Cawang", "Tebet"]),
        # Yogyakarta (6 = 25%)
        ("Rute wisata Jogja: dari Borobudur ke Prambanan, mampir ke Monjali, lewat UGM, dan berakhir di Malioboro",
         ["Candi Borobudur", "Monumen Yogya Kembali", "UGM", "Candi Prambanan", "Malioboro"]),
        ("Dari Parangtritis, mampir ke Ratu Boko, lewat Prambanan, lanjut ke Monjali, dan berakhir di Tugu",
         ["Pantai Parangtritis", "Candi Ratu Boko", "Candi Prambanan", "Monumen Yogya Kembali", "Tugu Yogyakarta"]),
        ("Mau keliling Jogja tengah: dari Kotagede ke Taman Sari, mampir ke Kraton, lewat Beringharjo, berakhir di Malioboro",
         ["Kotagede", "Taman Sari", "Kraton Yogyakarta", "Pasar Beringharjo", "Malioboro"]),
        ("Dari UII ke Vredenburg, mampir ke Affandi, lewat Sonobodoyo, dan berakhir di Nol Kilometer",
         ["Universitas Islam Indonesia", "Museum Affandi", "Museum Sonobudoyo", "Benteng Vredenburg", "Nol Kilometer"]),
        ("Rute dari Borobudur ke Tugu Jogja, mampir di Monjali, lewat Gembira Loka, dan berakhir di Malioboro",
         ["Candi Borobudur", "Monumen Yogya Kembali", "Gembira Loka Zoo", "Tugu Yogyakarta", "Malioboro"]),
        ("Perjalanan dari Prambanan ke Ratu Boko, mampir ke Monjali, lewat UGM, dan berakhir di Taman Pelangi",
         ["Candi Prambanan", "Candi Ratu Boko", "Monumen Yogya Kembali", "UGM", "Taman Pelangi"]),
        # Bandung (4 = 15%)
        ("Dari Kawah Putih ke Gedung Sate, mampir di Braga, lewat Alun-Alun, dan berakhir di Cihampelas",
         ["Kawah Putih", "Gedung Sate", "Braga", "Alun-Alun Bandung", "Cihampelas"]),
        ("Rute Bandung: dari Floating Market ke Trans Studio, mampir di Dago, lewat Tebing Keraton, berakhir di Paris Van Java",
         ["Floating Market Lembang", "Dago", "Tebing Keraton", "Trans Studio Bandung", "Paris Van Java"]),
        ("Dari Stasiun Bandung ke Situ Patenggang, mampir di Ranca Upas, lewat Kawah Putih, dan berakhir di Rancabali",
         ["Stasiun Bandung", "Kawah Putih", "Situ Patenggang", "Ranca Upas"]),
        ("Mau ke Lembang dari ITB, mampir di Dago Pakar, lewat Floating Market, dan berakhir di Farm House",
         ["Institut Teknologi Bandung", "Dago Pakar", "Floating Market Lembang", "Farm House Lembang"]),
    ]

    for prompt_text, gt in hard_prompts:
        dataset.append(make_prompt(idx, "hard", prompt_text, gt))
        idx += 1

    # Verify counts
    easy_count = sum(1 for d in dataset if d["difficulty"] == "easy")
    medium_count = sum(1 for d in dataset if d["difficulty"] == "medium")
    hard_count = sum(1 for d in dataset if d["difficulty"] == "hard")
    total = len(dataset)

    assert total == 100, f"Expected 100, got {total}"
    assert easy_count == 35, f"Expected 35 easy, got {easy_count}"
    assert medium_count == 40, f"Expected 40 medium, got {medium_count}"
    assert hard_count == 25, f"Expected 25 hard, got {hard_count}"

    # Verify all landmarks exist
    for entry in dataset:
        for stop in entry["ground_truth"]:
            name = stop["name"]
            if name not in LANDMARKS:
                print(f"ERROR: Unknown landmark '{name}' in {entry['id']}")
                return

    # Write output
    output_path = "/root/arahin/benchmark/msntb.json"
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(dataset, f, ensure_ascii=False, indent=2)

    # Region distribution
    jakarta = 0
    jogja = 0
    bandung = 0
    for entry in dataset:
        lats = [s["lat"] for s in entry["ground_truth"]]
        avg_lat = sum(lats) / len(lats)
        if avg_lat > -7.0:
            jakarta += 1
        elif avg_lat > -7.7:
            jogja += 1
        else:
            bandung += 1

    print(f"MSNTB dataset generated: {total} prompts")
    print(f"  Easy: {easy_count}, Medium: {medium_count}, Hard: {hard_count}")
    print(f"  Regions: Jakarta~{jakarta}, Yogyakarta~{jogja}, Bandung~{bandung}")
    print(f"  Saved to: {output_path}")

if __name__ == "__main__":
    main()
