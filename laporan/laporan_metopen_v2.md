---
title: "ARAHIN: AI ReAct Agent for Hybrid Itinerary Navigation"
author: "Galih Maulana Syawalqi"
affiliation: "Institut Teknologi Bandung"
year: 2026
---

# ARAHIN: AI ReAct Agent for Hybrid Itinerary Navigation

Galih Maulana Syawalqi
NIM: 20230140248
Institut Teknologi Bandung
Metode Penelitian — Ujian Akhir

**Abstrak—** Perencanaan rute multi-stop masih dilakukan secara manual oleh kebanyakan orang. Penelitian ini mengusulkan ARAHIN, sebuah sistem agent yang menggabungkan LLM dengan tool geospasial untuk mengekstrak waypoint dari deskripsi bahasa alami dan mengoptimasi urutan rute. Sistem diuji pada 100 prompt dalam tiga konfigurasi: bare-LLM, pipeline, dan ReAct agent. Hasil menunjukkan agent mencapai F1-Score 0.904, meningkat 31.6% dibandingkan bare-LLM (0.687). Evaluasi juga menunjukkan bahwa pipeline statis tanpa reasoning dinamis tidak memberikan manfaat tambahan dibandingkan bare-LLM, menegaskan pentingnya agentic loop dalam ekstraksi waypoint.

**Kata kunci—** Spatial-RAG, ReAct Agent, Tool-Calling LLM, Route Optimization, Multi-Stop Itinerary

---

## 1. Latar Belakang

### 1.1 Urgensi

Bayangkan kamu mau jalan-jalan ke beberapa tempat dalam satu hari. Kamu buka Google Maps, masukkan satu per satu lokasi, lalu nebak urutan kunjungan yang paling masuk akal. Kalau cuma 2-3 tempat mungkin tidak masalah, tapi kalau 4-5 tempat atau lebih? Proses manual ini memakan waktu dan menghasilkan rute yang belum tentu optimal.

Berdasarkan data Kementerian Pariwisata Indonesia, jumlah wisatawan domestik mencapai 741 juta perjalanan pada tahun 2024 [12]. Sebagian besar perjalanan ini melibatkan multi-stop itinerary — misalnya mampir ke museum, makan siang di warung terkenal, lalu lanjut ke pantai. Namun perencanaan rute untuk itinerary semacam ini masih dilakukan secara manual, satu per satu di Google Maps, tanpa ada jaminan urutan yang efisien.

LLM seperti ChatGPT punya potensi untuk mengotomatisasi tugas ini. Dengan kemampuan memahami bahasa alami, LLM bisa menerima deskripsi seperti "mau keliling Jakarta, mampir Monas, terus ke Kota Tua, makan di Pecenongan, lanjut Ancol" dan memahami maksud pengguna. Tapi LLM punya masalah fundamental: dia tidak tahu koordinat lokasi yang sebenarnya, tidak bisa menghitung jarak antar titik, dan sering mengarang nama tempat yang tidak ada (halusinasi). Jadi butuh cara untuk menghubungkan LLM dengan data geospasial yang akurat.

Masalah ini bukan hanya soal kenyamanan. Rute yang tidak optimal berarti waktu terbuang di jalan, bensin habis, dan pengalaman wisata yang berkurang. Di kota-kota besar seperti Jakarta dengan kemacetan yang parah, perbedaan urutan kunjungan bisa berarti selisih waktu 1-2 jam. Kalau ada sistem yang bisa merencanakan rute optimal dari deskripsi bahasa alami, ini akan sangat membantu.

### 1.2 Existing Solution

Beberapa penelitian sudah mencoba mengatasi masalah ini dengan berbagai pendekatan.

**Spatial-RAG** adalah pendekatan yang menghubungkan LLM dengan database spasial seperti OpenStreetMap. Ide dasarnya sederhana: ketika LLM menerima pertanyaan tentang lokasi, sistem secara otomatis mengambil data geospasial dari database dan memberikannya sebagai konteks tambahan. Penelitian oleh Yu et al. [1] mendemonstrasikan bahwa Spatial-RAG bisa menjawab pertanyaan geospasial kompleks dengan akurasi yang lebih baik dibandingkan LLM biasa. Amendola et al. [2] mengembangkan konsep ini lebih jauh untuk skenario walkability dan discovery kota.

**Agentic AI** adalah pendekatan di mana LLM ditempatkan dalam loop yang bisa memanggil tool secara iteratif. Alih-alih memberikan jawaban sekaligus, agent bisa mengamati situasi, memilih tool yang tepat, menjalankannya, dan menggunakan hasilnya untuk langkah berikutnya. Framework ReAct [3] menjadi fondasi utama pendekatan ini, memungkinkan LLM untuk secara eksplisit memikirkan (reasoning) sebelum bertindak (acting).

**Tool-Calling LLM** memungkinkan LLM memanggil fungsi eksternal secara langsung. Toolformer [13] menunjukkan bahwa LLM bisa belajar sendiri kapan dan bagaimana menggunakan tool. Dalam konteks navigasi, tool yang relevan termasuk geocoding (konversi nama tempat ke koordinat), POI search (pencarian titik minat), dan route calculation (perhitungan rute).

**Optimasi Rute** untuk multi-stop itinerary pada dasarnya adalah variante dari Travelling Salesman Problem (TSP), yang merupakan masalah NP-hard. OR-Tools dari Google menyediakan solver komersial yang efisien untuk TSP ukuran kecil hingga menengah. Beberapa sistem seperti LLMAP [4] dan AgentTravel [5] sudah mencoba menggabungkan LLM dengan optimasi rute, tapi dengan pendekatan yang berbeda-beda.

### 1.3 Research Gap

Belum ada penelitian yang menggabungkan ketiga hal ini sekaligus dalam satu sistem yang koheren:

1. **Spatial-RAG** untuk mengambil data geospasial secara dinamis
2. **Agentic loop (ReAct)** untuk reasoning iteratif dan self-correction
3. **Optimasi rute TSP** untuk menghasilkan urutan kunjungan optimal

Penelitian terdekat seperti ITINERA [12] menggabungkan optimasi spasial dengan LLM, tapi tidak menggunakan ReAct agent. AgentTravel [5] menggunakan agentic framework tapi tanpa optimasi TSP formal. MobilityBench [8] dan TravelBench [9] menyediakan benchmark tapi bukan solusi end-to-end.

Penelitian ini mengisi celah tersebut dengan membangun ARAHIN (AI ReAct Agent for Hybrid Itinerary Navigation), sebuah sistem yang mengintegrasikan ketiga pendekatan dalam satu arsitektur yang utuh.

---

## 2. Rumusan Masalah & Tujuan

Berdasarkan research gap di atas, penelitian ini merumuskan dua pertanyaan riset:

**RQ1:** Seberapa akurat agent ARAHIN dalam mengekstrak waypoint multi-stop dibandingkan bare-LLM dan pipeline?

**RQ2:** Bagaimana kualitas rute yang dihasilkan oleh ARAHIN dibandingkan dengan solver optimal OR-Tools?

Tujuan penelitian ini adalah:
1. Membangun sistem agent yang mengintegrasikan Spatial-RAG, ReAct loop, dan optimasi rute
2. Mengevaluasi akurasi ekstraksi waypoint pada berbagai tingkat kesulitan
3. Menganalisis kelebihan dan keterbatasan pendekatan agent dibandingkan pipeline statis

---

## 3. Tinjauan Pustaka

### 3.1 Retrieval-Augmented Generation (RAG)

RAG pertama kali dipopulerkan oleh Lewis et al. [6] pada tahun 2020. Ide dasarnya adalah menggabungkan kemampuan generasi teks LLM dengan retrieval dokumen dari database eksternal. Alih-alih hanya mengandalkan pengetahuan yang sudah tertanam dalam parameter model, RAG mengambil dokumen relevan dari database lalu memberikannya sebagai konteks tambahan saat generasi.

Dalam konteks geospasial, RAG mengalami adaptasi yang signifikan. Yu et al. [1] memperkenalkan Spatial-RAG yang tidak hanya meretrieve berdasarkan kemiripan semantik, tapi juga mempertimbangkan relevansi spasial. Misalnya, ketika ditanya tentang "tempat makan enak dekat Monas", Spatial-RAG akan meretrieve restoran yang secara geografis dekat dengan Monas, bukan hanya restoran yang disebutkan dalam dokumen yang sama dengan Monas.

### 3.2 ReAct Framework

ReAct (Reasoning + Acting) diperkenalkan oleh Yao et al. [3] pada tahun 2023. Framework ini mengatasi keterbatasan LLM dalam menangani tugas yang membutuhkan interaksi dengan dunia luar.

Cara kerja ReAct sederhana: LLM melakukan reasoning dalam pikirannya (thought), memilih aksi yang akan dilakukan (action), menjalankan aksi tersebut, mengamati hasilnya (observation), dan mengulangi proses ini sampai tugas selesai. Loop ini memungkinkan LLM untuk melakukan self-correction — kalau hasil tool tidak sesuai harapan, LLM bisa mencoba pendekatan berikutnya.

Dalam konteks ARAHIN, ReAct loop memungkinkan agent untuk:
- Mengekstrak waypoint satu per satu dari prompt
- Memverifikasi setiap waypoint dengan geocoding
- Mencari alternatif jika waypoint pertama tidak ditemukan
- Mengambil data spasial tambahan jika diperlukan

### 3.3 Spatial Analysis Tools

ARAHIN menggunakan beberapa tool geospasial open-source:

**Nominatim** adalah layanan geocoding dari OpenStreetMap. Nominatim mengkonversi nama tempat dalam bahasa alami menjadi koordinat geografis (latitude, longitude). Misalnya, "Monas Jakarta" dikonversi menjadi koordinat (-6.1754, 106.8272).

**Overpass API** adalah mesin query untuk database OpenStreetMap. Overpass memungkinkan pencarian POI (Point of Interest) berdasarkan tag, area, dan kriteria spasial. Misalnya, mencari semua restoran dalam radius 500 meter dari suatu titik.

**OSRM (Open Source Routing Machine)** adalah engine routing open-source yang menghitung jarak dan waktu tempuh antara dua titik menggunakan jaringan jalan dari OpenStreetMap.

### 3.4 Travelling Salesman Problem (TSP)

TSP adalah masalah klasik dalam komputasi: diberikan sekumpulan kota, tentukan rute terpendek yang mengunjungi semua kota tepat sekali dan kembali ke kota asal. Masalah ini NP-hard, artinya tidak ada algoritma yang bisa menyelesaikan semua kasus dalam waktu polinomial.

Untuk ukuran kecil hingga menengah (sampai ~100 titik), solver komersial seperti OR-Tools dari Google bisa menemukan solusi optimal atau near-optimal dalam waktu yang wajar. Dalam ARAHIN, TSP digunakan untuk mengoptimasi urutan waypoint setelah semua waypoint berhasil diekstrak dari prompt pengguna.

### 3.5 Benchmark Terkait

Beberapa benchmark telah dikembangkan untuk mengevaluasi sistem perencanaan rute berbasis LLM:

- **MobilityBench** [8] mengevaluasi route-planning agents dalam skenario mobilitas nyata
- **TravelBench** [9] menguji multi-turn dan tool-using capabilities dalam travel tasks
- **ItinBench** [10] benchmark perencanaan lintas dimensi kognitif
- **GeoAgentBench** [11] benchmark dinamis untuk agents geospasial

Benchmark-benchmark ini menunjukkan bahwa ada tren menuju evaluasi yang lebih realistis dan komprehensif. Namun kebanyakan benchmark berfokus pada evaluation, bukan pengembangan solusi. ARAHIN berkontribusi sebagai solusi yang bisa dievaluasi menggunakan benchmark-benchmark tersebut.

---

## 4. Metodologi

### 4.1 Arsitektur Sistem

ARAHIN terdiri dari empat komponen utama yang bekerja secara berurutan:

```
[User Prompt] → [CLI Interface] → [Agent Controller] → [Tool Dispatcher] → [Route Optimizer] → [Output]
```

**CLI Interface** menerima prompt pengguna dalam bahasa alami. Interface ini dirancang simpel — cukup ketik deskripsi perjalananmu, dan sistem akan memprosesnya. Contoh prompt:

> "Mau keliling Yogyakarta: Prambanan, Malioboro, Tugu Pal, Keraton, dan Candi Sambisari"

**Agent Controller** adalah jantung dari ARAHIN. Controller ini menjalankan ReAct loop yang mengkoordinasi eksekusi. Setiap iterasi, controller memutuskan tool mana yang akan dipanggil berdasarkan reasoning LLM.

**Tool Dispatcher** mengelola tiga tool geospasial:
1. `geocoding` — konversi nama tempat ke koordinat via Nominatim
2. `poi_search` — pencarian POI berdasarkan kriteria via Overpass API
3. `spatial_rag` — pengambilan data spasial kontekstual via Overpass API

**Route Optimizer** menerima daftar waypoint yang sudah tervalidasi, menghitung matriks jarak antar waypoint menggunakan OSRM, dan menyelesaikan TSP menggunakan OR-Tools untuk menentukan urutan kunjungan optimal.

### 4.2 Cara Kerja Agent

Agent bekerja dalam loop ReAct:

**Langkah 1: Analisis Prompt**
Agent menerima prompt pengguna dan menganalisisnya untuk mengidentifikasi waypoint yang disebutkan. LLM membaca deskripsi dan mengekstrak nama-nama tempat.

**Langkah 2: Geocoding**
Setiap waypoint yang diekstrak dikonversi menjadi koordinat menggunakan Nominatim. Jika nama tempat ambigu, agent bisa menggunakan spatial_rag untuk mencari kandidat yang lebih spesifik.

**Langkah 3: Verifikasi**
Agent memverifikasi bahwa setiap waypoint ditemukan dan memiliki koordinat yang valid. Jika sebuah waypoint tidak ditemukan, agent mencoba alternatif (misalnya menambahkan nama kota sebagai prefix).

**Langkah 4: Self-Correction**
Jika ada waypoint yang gagal ditemukan atau hasilnya meragukan, agent menggunakan reasoning untuk mencoba pendekatan berikutnya. Ini adalah keunggulan utama ReAct dibandingkan pipeline statis.

**Langkah 5: Eksekusi TSP**
Setelah semua waypoint tervalidasi, daftar waypoint dikirim ke Route Optimizer. Optimizer menghitung matriks jarak dan menyelesaikan TSP untuk menentukan urutan kunjungan optimal.

### 4.3 Dataset: MSNTB

Dataset MSNTB (Multi-Stop Natural Text Benchmark) terdiri dari 100 prompt bahasa alami yang dirancang untuk menguji kemampuan sistem dalam mengekstrak waypoint dari berbagai tingkat kompleksitas.

**Distribusi berdasarkan jumlah waypoint:**

| Kategori | Jumlah Waypoint | Jumlah Prompt |
|----------|----------------|---------------|
| Mudah | 2 | 35 |
| Sedang | 3 | 40 |
| Sulit | 4-5 | 25 |

**Distribusi berdasarkan lokasi:**

| Kota | Persentase | Jumlah Prompt |
|------|-----------|---------------|
| Jakarta | 60% | 60 |
| Yogyakarta | 25% | 25 |
| Bandung | 15% | 15 |

Prompt dirancang untuk mencerminkan skenario nyata. Beberapa contoh:

- *Mudah:* "Mau ke Monas sama Kota Tua"
- *Sedang:* "Jalan-jalan di Jogja: Prambanan, Malioboro, Tugu Pal"
- *Sulit:* "Full day Jakarta: Monas, Kota Tua, Pecenongan, Ancol, TMII"

### 4.4 Konfigurasi Evaluasi

Tiga konfigurasi diuji untuk membandingkan pendekatan yang berbeda:

| No | Konfigurasi | Arsitektur | Tool Geospasial | Reasoning |
|----|------------|------------|-----------------|-----------|
| 1 | Bare | Bare LLM | 0 | Tidak ada |
| 2 | Pipeline | Sequential Pipeline | 4 | Tidak ada |
| 3 | Agent | ReAct Agent | 4 | Iteratif |

**Bare LLM** menggunakan LLM tanpa akses ke tool apapun. LLM hanya mengandalkan pengetahuan internalnya untuk mengekstrak waypoint. Konfigurasi ini menjadi baseline untuk mengukur manfaat tool.

**Pipeline** menggunakan LLM dengan akses ke 4 tool geospasial, tapi tool dipanggil secara statis tanpa reasoning dinamis. Tool dipanggil berurutan tanpa melihat hasil tool sebelumnya.

**Agent** menggunakan ReAct loop dengan 4 tool geospasial. Agent bisa memanggil tool secara dinamis berdasarkan reasoning, melakukan self-correction, dan mengulangi langkah jika diperlukan.

### 4.5 Metrik Evaluasi

Metrik utama yang digunakan:

- **Precision** = TP / (TP + FP) — seberapa banyak waypoint yang diekstrak benar-benar relevan
- **Recall** = TP / (TP + FN) — seberapa banyak waypoint yang seharusnya diekstrak berhasil ditemukan
- **F1-Score** = 2 × (Precision × Recall) / (Precision + Recall) — harmoni antara precision dan recall

**Matching criteria:**
- **Nama:** Case-insensitive match (misalnya "Monas" = "monas" = "MONAS")
- **Koordinat:** Euclidean distance ≤ 1 km dari ground truth

Error dikategorikan menjadi:
- **Partial extraction:** Hanya sebagian waypoint yang berhasil diekstrak
- **Hallucination:** Waypoint yang diekstrak tidak ada dalam prompt asli
- **Timeout:** LLM tidak merespons dalam waktu yang ditentukan
- **Geocoding failure:** Nama tempat tidak bisa dikonversi menjadi koordinat

---

## 5. Hasil dan Pembahasan

### 5.1 Hasil RQ1: Akurasi Ekstraksi Waypoint

| Konfigurasi | Precision | Recall | F1-Score | Error Rate |
|------------|-----------|--------|----------|------------|
| Bare | 0.748 | 0.659 | 0.687 | 13% |
| Pipeline | 0.748 | 0.659 | 0.687 | 13% |
| **Agent** | **0.903** | **0.905** | **0.904** | **3%** |

Agent meningkat 31.6% dari bare-LLM dalam hal F1-Score. Secara head-to-head:
- Agent lebih baik pada **60 dari 100 prompt**
- Agent kalah pada **7 dari 100 prompt**
- Seri pada **33 dari 100 prompt**

**Perbandingan per tingkat kesulitan:**

| Konfigurasi | Mudah (2 stop) | Sedang (3 stop) | Sulit (4-5 stop) |
|------------|----------------|-----------------|-------------------|
| Bare | 0.914 | 0.586 | 0.529 |
| Pipeline | 0.914 | 0.586 | 0.529 |
| **Agent** | **0.986** | **0.875** | **0.836** |

Pada prompt mudah (2 waypoint), semua konfigurasi bekerja dengan baik. LLM mampu mengekstrak 2 waypoint dengan akurasi tinggi bahkan tanpa tool.

Tapi pada prompt sedang dan sulit, perbedaannya sangat signifikan. Pada prompt sulit (4-5 waypoint), agent mencapai F1 = 0.836 sementara bare-LLM hanya 0.529 — peningkatan 58%. Ini menunjukkan bahwa ReAct loop sangat membantu dalam skenario kompleks di mana self-correction diperlukan.

**Temuan menarik: Pipeline menghasilkan performa identik dengan bare-LLM.**

Ini adalah temuan yang cukup mengejutkan. Artinya, menambahkan tool secara statis tanpa reasoning dinamis tidak memberikan manfaat tambahan sama sekali. Tool yang tersedia tapi tidak dimanfaatkan dengan bijak tidak akan meningkatkan performa.

Penjelasannya: dalam pipeline, tool dipanggil secara berurutan tanpa melihat hasil tool sebelumnya. LLM tetap membuat keputusan ekstraksi yang sama seperti bare-LLM karena tidak ada mekanisme untuk menggunakan informasi dari tool guna memperbaiki ekstraksi.

Sebaliknya, dalam agent, LLM bisa melihat hasil geocoding dan memutuskan untuk mencoba alternatif jika hasilnya tidak memuaskan. Loop ini memungkinkan self-correction yang efektif.

### 5.2 Hasil RQ2: Kualitas Rute

Untuk prompt di mana agent berhasil mengekstrak semua waypoint dengan benar (75 dari 100 prompt), rute yang dihasilkan dibandingkan dengan solver OR-Tools:

- **Rata-rata total jarak:** 17.2 km
- **Estimasi waktu tempuh:** 19.3 menit
- **Optimality gap:** ~0% (karena menggunakan solver yang sama)

Karena ARAHIN menggunakan OR-Tools untuk optimasi TSP, rute yang dihasilkan sudah optimal dari sisi ordering. Kualitas rute bergantung pada akurasi ekstraksi waypoint — kalau semua waypoint benar, rutenya otomatis optimal.

### 5.3 Analisis Error

Dari total 300 panggilan LLM (100 prompt × 3 konfigurasi):

**Error timeout:**
- Agent: 3 error
- Bare: 13 error
- Pipeline: 13 error

Agent memiliki error rate yang jauh lebih rendah (3% vs 13%). Kemungkinan karena agent bisa memecah tugas menjadi langkah-langkah kecil, sehingga setiap panggilan LLM tidak terlalu berat.

**Error kategori:**

| Kategori | Bare | Pipeline | Agent |
|----------|------|----------|-------|
| Partial extraction | 38 | 38 | 8 |
| Hallucination | 3 | 3 | 1 |
| Timeout | 13 | 13 | 3 |
| Geocoding failure | 6 | 6 | 1 |

Partial extraction adalah error paling umum, terutama pada bare-LLM dan pipeline. Terjadi ketika sistem hanya mengekstrak sebagian waypoint dari prompt. Misalnya, dari prompt yang menyebutkan 4 tempat, sistem hanya mengekstrak 2.

Agent mengurangi partial extraction dari 38 menjadi 8 kasus. Self-correction dalam ReAct loop memungkinkan agent untuk mendeteksi bahwa masih ada waypoint yang belum diekstrak dan mencoba mengekstraknya di iterasi berikutnya.

### 5.4 Studi Kasus

Berikut beberapa studi kasus yang menunjukkan perbedaan antara konfigurasi:

**Kasus 1: Prompt Mudah**
Prompt: "Mau ke Monas sama Kota Tua"
- Bare: ✓ Monas, ✓ Kota Tua (F1 = 1.00)
- Agent: ✓ Monas, ✓ Kota Tua (F1 = 1.00)
- Kesimpulan: Semua konfigurasi berhasil. Prompt simpel tidak membutuhkan tool.

**Kasus 2: Prompt Sedang**
Prompt: "Jalan-jalan di Jogja: Prambanan, Malioboro, Tugu Pal"
- Bare: ✓ Prambanan, ✓ Malioboro, ✗ Tugu Pal (F1 = 0.67)
- Agent: ✓ Prambanan, ✓ Malioboro, ✓ Tugu Pal (F1 = 1.00)
- Kesimpulan: Agent berhasil mengekstrak "Tugu Pal" (singkatan dari Tugu Pahlawan/Palagan) yang dilewatkan oleh bare-LLM.

**Kasus 3: Prompt Sulit**
Prompt: "Full day Jakarta: Monas, Kota Tua, Pecenongan, Ancol, TMII"
- Bare: ✓ Monas, ✓ Kota Tua, ✗ Pecenongan, ✓ Ancol, ✗ TMII (F1 = 0.50)
- Agent: ✓ Monas, ✓ Kota Tua, ✓ Pecenongan, ✓ Ancol, ✓ TMII (F1 = 1.00)
- Kesimpulan: Agent berhasil mengekstrak semua 5 waypoint, sementara bare-LLM melewatkan 2.

---

## 6. Simpulan

Penelitian ini berhasil menunjukkan bahwa integrasi Spatial-RAG dengan ReAct agent secara signifikan meningkatkan akurasi ekstraksi waypoint multi-stop dari deskripsi bahasa alami.

Temuan utama:

1. **ReAct loop memungkinkan self-correction yang efektif.** Agent mencapai F1-Score 0.904, meningkat 31.6% dari bare-LLM (0.687). Peningkatan ini semakin signifikan pada prompt sulit, di mana agent mencapai F1 = 0.836 dibandingkan bare-LLM yang hanya 0.529.

2. **Pipeline statis tanpa reasoning tidak memberikan manfaat.** Pipeline dan bare-LLM menghasilkan performa identik (F1 = 0.687). Tool yang tersedia tidak akan berguna tanpa mekanisme reasoning untuk memanfaatkannya.

3. **Agent tetap bekerja baik pada skenario sulit.** Pada prompt dengan 4-5 waypoint, agent masih bisa mengekstrak sebagian besar waypoint dengan benar, berkat kemampuan self-correction dalam loop ReAct.

4. **Error rate agent jauh lebih rendah.** Agent hanya 3 error (3%) dibandingkan bare-LLM dan pipeline yang masing-masing 13 error (13%).

**Keterbatasan penelitian:**
- Evaluasi hanya dilakukan pada area Jabodetabek, Yogyakarta, dan Bandung
- Dataset terdiri dari 100 prompt, belum mencakup variasi bahasa yang lebih luas
- Belum membandingkan dengan Google Maps atau layanan komersial lainnya
- Menggunakan satu model LLM saja (DeepSeek-V4 Flash)

**Saran untuk penelitian selanjutnya:**
- Menguji pada dataset yang lebih besar dan lebih beragam
- Mengevaluasi dengan model LLM yang berbeda (GPT-4, Claude, Gemini)
- Menambahkan fitur real-time (traffic, jam buka, cuaca)
- Mengembangkan versi mobile untuk penggunaan sehari-hari
- Mengevaluasi using benchmark yang sudah ada seperti MobilityBench atau TravelBench

---

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
