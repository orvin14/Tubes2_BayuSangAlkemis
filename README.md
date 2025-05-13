![image](https://github.com/user-attachments/assets/e50bba60-cf6d-406b-85d3-084e15294531)

# Tubes2_BayuSangAlkemis
Tugas Besar 2 IF2211 Strategi Algoritma Semester 2 Tahun 2024/2025

## Deskripsi Umum
Proyek pencarian resep elemen Little Alchemy 2 menggunakan algoritma BFS dan DFS. Program akan menunjukkan resep dalam bentuk tree dengan leaf adalah elemen dasar.

### Anggota Kelompok
|            Nama             |      NIM      |
| --------------------------  | ------------- |
| Orvin Andika Ikhsan Abhista |    13523017   |
| Joel Hotlan Haris Siahaan   |    13523025   |
| Fajar Kurniawan             |    13523027   |

## Fitur
* **Algoritma BFS** <br>
Pencarian resep elemen Little Alchemy 2 menggunakan algoritma BFS
* **Algoritma DFS** <br>
Pencarian resep elemen Little Alchemy 2 menggunakan algoritma DFS
* **Searching** <br>
Pencarian elemen Little Alchemy 2 berdasarkan nama
* **Tree** <br>
Menampilkan resep dalam bentuk tree


## Cara Menjalankan Program 
1. Buka alamat https://recipe-finder-production-6d64.up.railway.app/
2. Pilih mode (single atau multiple)
3. Pilih algoritma
4. Tekan elemen yang diingikan
5. Akan muncul sejumlah resep sesuai yang direquest seperti ini
   ![image](https://github.com/user-attachments/assets/f6f01f67-2fea-4b59-b6f7-cb9bc801b8bd)

## Cara Kerja BFS
1. Telusuri semua kemungkinan resep untuk membuat elemen target, masing-masing kemungkinan dimasukkan ke dalam sebuah state yang dipush ke queue of recipe state, kedua (atau salah satu) ingredients penyusunnya kemudian dimasukkan ke dalam queue of element di masing-masing state
2. Setiap state terdiri dari map untuk menyimpan kombinasi resep yang sudah ditemukan sejauh ini, (misal `Brick:[Mud, Fire], Mud:[Water, Soil]`) dan queue untuk menyimpan elemen yang selanjutnya harus diexpand untuk stat tersebut (misal `[Bread, Vegetables]`)
3. Untuk masing-masing state, akan di-expand setiap elemen dalam queue internal state tersebut (queue of element), kemudian dicari kemungkinan resep untuk masing-masing, untuk setiap variasi resep, kita duplikat state saat ini dan menambahkan resep dari elemen yang diexpand ke dalam recipeMap milik state tersebut dan menambahkan elemen ke queue elemen milik state tersebut (jika ada yang bisa dipush)
4. Setiap elemen pada queue internal pada setiap state akan diproses hingga queue kosong. Jika queue kosong berarti resep sudah jadi dan bisa dipush ke slice/list result.
5. Semua state pada recipeQueue akan diproses hingga queue kosong atau jumlah resep mencapai maxRecipe.
6. Diterapkan beberapa batasan yang mengurangi kemampuan BFS tetapi diperlukan karena penggunaan space memory tumbuh sangat cepat.
