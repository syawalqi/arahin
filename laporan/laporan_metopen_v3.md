---
title: "Laporan Ujian Akhir Metode Penelitian"
author: "Galih Maulana Syawalqi"
nim: "20230140248"
affiliation: "Institut Teknologi Bandung"
course: "Metode Penelitian"
year: 2026
---

# Laporan Ujian Akhir Metode Penelitian

**Nama:** Galih Maulana Syawalqi
**NIM:** 20230140248
**Mata Kuliah:** Metode Penelitian

---

## 1. Latar Belakang

Pernah nggak sih kamu mau jalan-jalan ke beberapa tempat dalam satu hari, tapi bingung urutan kunjungannya gimana yang paling efisien? Misalnya mau ke Monas, Kota Tua, makan di Pecenongan, terus lanjut Ancol. Kalau cuma 2-3 tempat mungkin masih bisa nebak, tapi kalau 4-5 tempat? Proses manual di Google Maps itu memakan waktu dan belum tentu optimal.

Saya coba hitung: kalau mau kunjungi 5 tempat, kamu harus buka Google Maps, masukkan satu per satu lokasi, cek jarak antar tempat, lalu nebak urutan yang paling masuk akal. Belum lagi kalau ada tempat yang namanya mirip (misalnya ada "Tugu Pal" dan "Tugu Pahlawan"), kamu harus mastikan dulu yang mana yang dimaksud. Proses ini bisa makan waktu 15-20 menit, dan hasilnya belum tentu optimal.

Dari masalah inilah saya tertarik untuk membuat sistem yang bisa membantu merencanakan rute multi-stop secara otomatis. LLM seperti ChatGPT sudah punya kemampuan memahami bahasa alami, jadi kalau kita bisa menghubungkannya dengan data geospasial yang akurat, hasilnya bisa sangat membantu.

Tapi ada masalah: LLM itu sering mengarang nama tempat yang nggak ada (halusinasi), nggak tahu koordinat lokasi, dan nggak bisa menghitung jarak. Jadi saya perlu cara untuk menghubungkan LLM dengan data peta yang benar.

Selain itu, ada tren yang menarik di bidang AI research. Banyak penelitian yang mencoba menggabungkan LLM dengan tool eksternal, tapi kebanyakan fokus pada domain tertentu seperti coding atau QA. Belum banyak yang fokus pada navigasi dan perencanaan rute. Padahal ini masalah sehari-hari yang dialami banyak orang.

Dari situlah saya mengembangkan ARAHIN (AI ReAct Agent for Hybrid Itinerary Navigation), sebuah sistem yang menggabungkan LLM dengan tool geospasial untuk mengekstrak waypoint dari deskripsi bahasa alami dan mengoptimasi urutan rute.

## 2. Rumusan Masalah & Tujuan

Dari latar belakang di atas, saya merumuskan dua pertanyaan riset:

**RQ1:** Seberapa akurat sistem ARAHIN dalam mengekstrak waypoint multi-stop dibandingkan LLM biasa?

**RQ2:** Bagaimana kualitas rute yang dihasilkan dibandingkan solver optimal?

**Tujuan penelitian:**
1. Membangun sistem agent yang mengintegrasikan LLM dengan tool geospasial
2. Mengevaluasi akurasi ekstraksi waypoint pada berbagai tingkat kesulitan
3. Menganalisis kelebihan dan keterbatasan pendekatan agent

## 3. Tinjauan Pustaka

### 3.1 Retrieval-Augmented Generation (RAG)

RAG itu konsep di mana LLM nggak cuma mengandalkan pengetahuan dari training, tapi juga mengambil informasi tambahan dari database eksternal. Lewis et al. [6] mempopulerkan konsep ini tahun 2020.

Konsep dasarnya sederhana: ketika LLM menerima pertanyaan, sistem secara otomatis mengambil dokumen relevan dari database lalu memberikannya sebagai konteks tambahan saat generasi. Jadi LLM punya "catatan" tambahan yang bisa dirujuk.

Dalam konteks geospasial, Spatial-RAG [1] mengembangkan ide ini lebih jauh. Misalnya, kalau kamu tanya "tempat makan enak dekat Monas", Spatial-RAG akan mengambil data restoran yang memang secara geografis dekat Monas, bukan cuma restoran yang sering disebut bareng Monas di internet. Pendekatan ini mempertimbangkan relevansi spasial, bukan cuma kemiripan semantik.

### 3.2 ReAct Framework

ReAct (Reasoning + Acting) diperkenalkan oleh Yao et al. [3] tahun 2023. Konsepnya sederhana: LLM berpikir dulu (reasoning), baru bertindak (acting). Prosesnya berulang:
1. LLM berpikir tentang apa yang perlu dilakukan
2. LLM memilih tool yang akan dipanggil
3. Tool dijalankan
4. LLM melihat hasilnya
5. Ulangi sampai tugas selesai

Loop ini memungkinkan LLM melakukan self-correction. Kalau hasil tool nggak sesuai harapan, LLM bisa coba pendekatan lain di iterasi berikutnya. Ini beda banget dengan pendekatan statis di mana semua langkah sudah ditentukan sebelumnya.

### 3.3 Tool Geospasial

ARAHIN menggunakan beberapa tool open-source:

- **Nominatim:** Konversi nama tempat jadi koordinat. Misalnya "Monas Jakarta" jadi (-6.1754, 106.8272). Tool ini pakai data dari OpenStreetMap
- **Overpass API:** Mesin query untuk database OpenStreetMap. Bisa cari POI berdasarkan tag dan area. Misalnya cari semua restoran dalam radius 500 meter
- **OSRM:** Engine routing open-source. Menghitung jarak dan waktu tempuh antara dua titik pakai jaringan jalan dari OpenStreetMap

### 3.4 Travelling Salesman Problem (TSP)

TSP itu masalah klasik: diberikan beberapa kota, tentukan rute terpendek yang mengunjungi semua kota tepat sekali dan kembali ke kota asal. Masalah ini NP-hard, tapi untuk ukuran kecil sampai menengah (sampai ~100 titik), solver seperti OR-Tools dari Google bisa menyelesaikan dalam waktu wajar.

Di ARAHIN, TSP digunakan untuk mengoptimasi urutan waypoint setelah semua waypoint berhasil diekstrak dari prompt pengguna. Jadi kalau user minta ke 5 tempat, optimizer akan menentukan urutan kunjungan yang paling efisien.

### 3.5 Benchmark Terkait

Beberapa benchmark sudah ada untuk evaluasi sistem perencanaan rute:
- MobilityBench [8] untuk skenario mobilitas nyata
- TravelBench [9] untuk multi-turn travel tasks
- ItinBench [10] untuk planning lintas dimensi kognitif
- GeoAgentBench [11] untuk agents geospasial

Benchmark-benchmark ini menunjukkan tren evaluasi yang makin realistis. Tapi kebanyakan fokus pada evaluation, bukan pengembangan solusi. ARAHIN berkontribusi sebagai solusi yang bisa dievaluasi pakai benchmark-benchmark tersebut.

## 4. Metodologi

### 4.1 Arsitektur Sistem

ARAHIN terdiri dari empat komponen:

```
[User Prompt] → [CLI Interface] → [Agent Controller] → [Tool Dispatcher] → [Route Optimizer] → [Output]
```

**CLI Interface:** Menerima prompt pengguna dalam bahasa alami. Cukup ketik deskripsi perjalanan, sistem akan memprosesnya. Contoh prompt:
> "Mau keliling Yogyakarta: Prambanan, Malioboro, Tugu Pal, Keraton, dan Candi Sambisari"

**Agent Controller:** Menjalankan ReAct loop. Setiap iterasi, controller memutuskan tool mana yang akan dipanggil berdasarkan reasoning LLM.

**Tool Dispatcher:** Mengelola tiga tool geospasial (geocoding, poi_search, spatial_rag).

**Route Optimizer:** Menerima daftar waypoint yang sudah tervalidasi, menghitung matriks jarak via OSRM, dan menyelesaikan TSP via OR-Tools untuk menentukan urutan kunjungan optimal.

### 4.2 Cara Kerja Agent

Saya mendesain agent bekerja dalam lima langkah:

**Langkah 1: Analisis Prompt**
Agent menerima prompt dan menganalisis untuk mengidentifikasi waypoint. LLM membaca deskripsi dan mengekstrak nama-nama tempat yang disebutkan.

**Langkah 2: Geocoding**
Setiap waypoint dikonversi ke koordinat pakai Nominatim. Kalau nama ambigu, agent pakai spatial_rag untuk cari kandidat lebih spesifik.

**Langkah 3: Verifikasi**
Agent memverifikasi setiap waypoint ditemukan dan punya koordinat valid. Kalau nggak ketemu, agent coba alternatif (misalnya menambahkan nama kota sebagai prefix).

**Langkah 4: Self-Correction**
Kalau ada waypoint gagal atau hasil meragukan, agent pakai reasoning untuk coba pendekatan lain. Ini keunggulan utama ReAct dibanding pipeline statis.

**Langkah 5: Eksekusi TSP**
Setelah semua waypoint tervalidasi, daftar waypoint dikirim ke Route Optimizer untuk optimasi urutan.

### 4.3 Dataset

Saya membuat dataset MSNTB (Multi-Stop Natural Text Benchmark) terdiri dari 100 prompt bahasa alami. Prompt dirancang untuk mencerminkan skenario nyata.

**Distribusi jumlah waypoint:**
- Mudah (2 waypoint): 35 prompt
- Sedang (3 waypoint): 40 prompt
- Sulit (4-5 waypoint): 25 prompt

**Distribusi lokasi:**
- Jakarta: 60%
- Yogyakarta: 25%
- Bandung: 15%

Contoh prompt:
- *Mudah:* "Mau ke Monas sama Kota Tua"
- *Sedang:* "Jalan-jalan di Jogja: Prambanan, Malioboro, Tugu Pal"
- *Sulit:* "Full day Jakarta: Monas, Kota Tua, Pecenongan, Ancol, TMII"

### 4.4 Konfigurasi Evaluasi

Saya menguji tiga konfigurasi untuk membandingkan pendekatan yang berbeda:

| No | Konfigurasi | Arsitektur | Tool |
|----|------------|------------|------|
| 1 | Bare | Bare LLM | 0 |
| 2 | Pipeline | Sequential Pipeline | 4 |
| 3 | Agent | ReAct Agent | 4 |

**Bare:** LLM tanpa akses tool. Baseline untuk mengukur manfaat tool.

**Pipeline:** LLM dengan 4 tool, tapi dipanggil statis tanpa reasoning dinamis. Tool dipanggil berurutan tanpa melihat hasil tool sebelumnya.

**Agent:** ReAct loop dengan 4 tool. Bisa memanggil tool dinamis berdasarkan reasoning, melakukan self-correction, dan mengulangi langkah jika diperlukan.

### 4.5 Metrik

- **Precision** = TP / (TP + FP) — seberapa banyak waypoint yang diekstrak benar-benar relevan
- **Recall** = TP / (TP + FN) — seberapa banyak waypoint yang seharusnya diekstrak berhasil ditemukan
- **F1-Score** = 2 × (Precision × Recall) / (Precision + Recall) — harmoni antara precision dan recall

Matching pakai nama (case-insensitive) atau koordinat (≤1km dari ground truth).

## 5. Hasil dan Pembahasan

### 5.1 Hasil RQ1

| Konfigurasi | Precision | Recall | F1-Score | Error |
|------------|-----------|--------|----------|-------|
| Bare | 0.748 | 0.659 | 0.687 | 13% |
| Pipeline | 0.748 | 0.659 | 0.687 | 13% |
| **Agent** | **0.903** | **0.905** | **0.904** | **3%** |

Agent meningkat 31.6% dari bare-LLM. Secara head-to-head:
- Agent lebih baik: 60/100 prompt
- Agent kalah: 7/100 prompt
- Seri: 33/100 prompt

**Per tingkat kesulitan:**

| Konfigurasi | Mudah (2 stop) | Sedang (3 stop) | Sulit (4-5 stop) |
|------------|----------------|-----------------|-------------------|
| Bare | 0.914 | 0.586 | 0.529 |
| Pipeline | 0.914 | 0.586 | 0.529 |
| **Agent** | **0.986** | **0.875** | **0.836** |

Pada prompt mudah, semua konfigurasi bekerja baik. LLM mampu mengekstrak 2 waypoint dengan akurasi tinggi bahkan tanpa tool.

Tapi pada prompt sedang dan sulit, perbedaannya sangat signifikan. Pada prompt sulit (4-5 waypoint), agent mencapai F1 = 0.836 sementara bare-LLM hanya 0.529 — peningkatan 58%. Ini menunjukkan bahwa ReAct loop sangat membantu dalam skenario kompleks di mana self-correction diperlukan.

**Temuan menarik:** Pipeline dan bare-LLM menghasilkan performa identik! Artinya, menambahkan tool tanpa reasoning nggak memberikan manfaat tambahan. Tool yang tersedia tapi nggak dimanfaatkan dengan bijak tetap nggak akan meningkatkan performa.

Penjelasannya: dalam pipeline, tool dipanggil secara berurutan tanpa melihat hasil tool sebelumnya. LLM tetap membuat keputusan ekstraksi yang sama seperti bare-LLM karena nggak ada mekanisme untuk menggunakan informasi dari tool guna memperbaiki ekstraksi.

Sebaliknya, dalam agent, LLM bisa melihat hasil geocoding dan memutuskan untuk mencoba alternatif kalau hasilnya nggak memuaskan. Loop ini memungkinkan self-correction yang efektif.

### 5.2 Hasil RQ2

Untuk prompt di mana agent berhasil mengekstrak semua waypoint (75 dari 100), rute yang dihasilkan dibandingkan dengan solver OR-Tools:

- Rata-rata total jarak: 17.2 km
- Estimasi waktu tempuh: 19.3 menit
- Optimality gap: ~0% (karena pakai solver yang sama)

Karena ARAHIN menggunakan OR-Tools untuk optimasi TSP, rute yang dihasilkan sudah optimal dari sisi ordering. Kualitas rute bergantung pada akurasi ekstraksi waypoint — kalau semua waypoint benar, rutenya otomatis optimal.

### 5.3 Analisis Error

Dari 300 panggilan LLM:

**Error timeout:**
- Agent: 3 error
- Bare/Pipeline: 13 error masing-masing

Agent memiliki error rate jauh lebih rendah (3% vs 13%). Kemungkinan karena agent memecah tugas jadi langkah-langkah kecil, jadi setiap panggilan LLM nggak terlalu berat.

**Error kategori:**

| Kategori | Bare | Pipeline | Agent |
|----------|------|----------|-------|
| Partial extraction | 38 | 38 | 8 |
| Hallucination | 3 | 3 | 1 |
| Timeout | 13 | 13 | 3 |
| Geocoding failure | 6 | 6 | 1 |

Partial extraction adalah error paling umum, terutama pada bare-LLM dan pipeline. Terjadi ketika sistem cuma mengekstrak sebagian waypoint dari prompt. Misalnya, dari prompt yang menyebutkan 4 tempat, sistem hanya mengekstrak 2.

Agent mengurangi partial extraction dari 38 jadi 8 kasus. Self-correction dalam ReAct loop memungkinkan agent untuk mendeteksi bahwa masih ada waypoint yang belum diekstrak dan mencoba mengekstraknya di iterasi berikutnya.

### 5.4 Studi Kasus

**Kasus 1: Prompt Mudah**
Prompt: "Mau ke Monas sama Kota Tua"
- Bare: ✓ Monas, ✓ Kota Tua (F1 = 1.00)
- Agent: ✓ Monas, ✓ Kota Tua (F1 = 1.00)
- Simpulan: Semua konfigurasi berhasil. Prompt simpel nggak butuh tool.

**Kasus 2: Prompt Sedang**
Prompt: "Jalan-jalan di Jogja: Prambanan, Malioboro, Tugu Pal"
- Bare: ✓ Prambanan, ✓ Malioboro, ✗ Tugu Pal (F1 = 0.67)
- Agent: ✓ Prambanan, ✓ Malioboro, ✓ Tugu Pal (F1 = 1.00)
- Simpulan: Agent berhasil mengekstrak "Tugu Pal" yang dilewatkan bare-LLM. Mungkin karena "Tugu Pal" adalah singkatan yang kurang umum, jadi bare-LLM nggak yakin.

**Kasus 3: Prompt Sulit**
Prompt: "Full day Jakarta: Monas, Kota Tua, Pecenongan, Ancol, TMII"
- Bare: ✓ Monas, ✓ Kota Tua, ✗ Pecenongan, ✓ Ancol, ✗ TMII (F1 = 0.50)
- Agent: ✓ Monas, ✓ Kota Tua, ✓ Pecenongan, ✓ Ancol, ✓ TMII (F1 = 1.00)
- Simpulan: Agent berhasil semua 5 waypoint, bare-LLM melewatkan 2. "Pecenongan" mungkin kurang dikenal LLM, dan "TMII" adalah akronim yang perlu dipecah.

## 6. Simpulan

Dari penelitian ini, saya mendapat beberapa temuan:

1. **ReAct loop sangat membantu.** Agent mencapai F1-Score 0.904, meningkat 31.6% dari bare-LLM (0.687). Peningkatan makin signifikan pada prompt sulit (58%).

2. **Pipeline statis nggak ada gunanya.** Pipeline dan bare-LLM menghasilkan performa identik. Tool yang tersedia tanpa reasoning tetap nggak akan membantu.

3. **Agent tetap bekerja baik pada skenario sulit.** Pada prompt 4-5 waypoint, agent masih bisa mengekstrak sebagian besar waypoint dengan benar.

4. **Error rate agent jauh lebih rendah.** Cuma 3 error (3%) dibanding bare-LLM 13 error (13%).

**Keterbatasan:**
- Evaluasi cuma di Jabodetabek, Yogyakarta, dan Bandung
- Dataset 100 prompt, belum luas
- Belum dibandingkan dengan Google Maps
- Pakai satu model LLM saja (DeepSeek-V4 Flash)

**Saran penelitian selanjutnya:**
- Menguji dataset lebih besar dan beragam
- Pakai model LLM berbeda (GPT-4, Claude, Gemini)
- Tambah fitur real-time (traffic, jam buka, cuaca)
- Kembangkan versi mobile
- Evaluasi pakai benchmark yang sudah ada

## Daftar Pustaka

[1] D. Yu, R. Bao, R. Ning, J. Peng, G. Mai, and L. Zhao, "Spatial-RAG: Spatial Retrieval Augmented Generation for Real-World Geospatial Reasoning Questions," arXiv:2502.18470, 2025.

[2] M. Amendola, C. P. Renso, and C. Renso, "Spatially-Enhanced Retrieval-Augmented Generation for Walkability and Urban Discovery," arXiv:2512.04790, 2025.

[3] S. Yao, J. Zhao, D. Yu, N. Du, I. Shafran, K. Narasimhan, and Y. Cao, "ReAct: Synergizing Reasoning and Acting in Language Models," arXiv:2210.03629, 2023.

[4] L. Yuan, D. Han, C. G. Brinton, and S. Brunswicker, "LLMAP: LLM-Assisted Multi-Objective Route Planning with User Preferences," arXiv:2509.12273, 2025.

[5] J. Zhao, J. Feng, and Y. Li, "AgentTravel: Knowledge-Augmented LLM Agent Framework for Urban Travel Planning," Proc. NeurIPS Workshop NORA, 2025.

[6] P. Lewis, E. Perez, A. Piktus, F. Petroni, V. Karpukhin, N. Goyal, H. Kuttler, M. Lewis, W. Yih, T. Rocktaschel, S. Riedel, and D. Kiela, "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks," arXiv:2005.11401, 2020.

[7] R. Bao, C. Yang, D. Yu, Z. Tang, G. Mai, and L. Zhao, "Spatial-Agent: Agentic Geo-spatial Reasoning with Scientific Core Concepts," arXiv:2601.16965, 2026.

[8] Z. Song, J. Zhang, C. Qin, C. Wang, C. Chen, L. Xu, K. Liu, X. Chu, and H. Zhu, "MobilityBench: A Benchmark for Evaluating Route-Planning Agents in Real-World Mobility Scenarios," arXiv:2602.22638, 2026.

[9] X. Cheng, Y. Hu, X. Zhang, L. Xu, L. Tan, Z. Pan, X. Li, and Y. Liu, "TravelBench: Beyond Itinerary Planning — A Real-World Benchmark for Multi-Turn and Tool-Using Travel Tasks," arXiv:2512.22673, 2025.

[10] T. Wang, P. Wang, W. Shi, and S. Li, "ItinBench: Benchmarking Planning Across Multiple Cognitive Dimensions with Large Language Models," arXiv:2603.19515, 2026.

[11] B. Yu, C. Yang, D. Hou, C. Liu, J. Liu, C. Wang, Z. Zhang, H. Li, and W. Yang, "GeoAgentBench: A Dynamic Execution Benchmark for Tool-Augmented Agents in Spatial Analysis," arXiv:2604.13888, 2026.

[12] Y. Tang, Z. Wang, A. Qu, Y. Yan, Z. Wu, D. Zhuang, J. Kai, K. Hou, X. Guo, J. Zhao, et al., "ITINERA: Integrating Spatial Optimization with Large Language Models for Open-domain Urban Itinerary Planning," Proc. EMNLP Industry Track, pp. 1413–1432, 2024.

[13] T. Schick, J. Dwivedi-Yu, R. Dessi, R. Raileanu, M. Lomeli, E. Hambro, L. Zettlemoyer, N. Cancedda, and T. Scialom, "Toolformer: Language Models Can Teach Themselves to Use Tools," arXiv:2302.04761, 2023.

[14] D. N. Campo, J. Conde, A. Alonso, G. Huecas, J. Salvachua, and P. Reviriego, "Real-time Spatial Retrieval Augmented Generation for Urban Environments," arXiv:2505.02271, 2025.

[15] W. Liang, M. Yuksekgonul, Y. Mao, E. Wu, and J. Zou, "GPT Detectors Are Biased against Non-Native English Writers," Patterns, Vol. 4, 2023.
