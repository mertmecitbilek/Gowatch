# GoWatch 🚀

GoWatch, kendi sunucunda barındırabileceğin (self-hosted), hafif ve modern bir web sitesi izleme aracıdır (uptime tracker). Go (Golang) ile geliştirilmiştir. Çoklu kullanıcı (multi-tenant) mimarisi sayesinde farklı kullanıcıların sisteme kayıt olup kendi servislerini bağımsızca izlemesine olanak tanır.

## Özellikler ✨

- **Çoklu Kullanıcı Mimarisi:** Kullanıcılar kayıt olabilir, giriş yapabilir ve sadece kendi monitörlerini ve bildirimlerini yönetebilir.
- **Farklı İzleme Türleri:** HTTP/HTTPS, TCP Port, Ping ve DNS sorgularını destekler.
- **Gerçek Zamanlı Durum (Heartbeats):** Geçmiş uptime (çalışma süresi) yüzdelerini ve gecikme (latency) sürelerini saklar.
- **Bildirim Kanalı:** Servislerden biri çöktüğünde anında haberiniz olur (Telegram).
- **Hafif ve Hızlı:** Go (Golang) ve SQLite ile güçlendirilmiştir. Sunucuyu yormaz, minimum kaynak tüketir.

---

## 🛠 Kurulum ve Çalıştırma (Docker ile)

GoWatch'ı herhangi bir bilgisayarda veya sunucuda çalıştırmanın en kolay yolu Docker kullanmaktır. Sadece **Docker** ve **Docker Compose**'un yüklü olması yeterlidir.

### 1. Projeyi İndirin

Bu proje klasörünün tamamını çalıştıracağınız makineye kopyalayın veya indirin.

### 2. Uygulamayı Başlatın

Terminalinizi (veya komut satırını) açın, `docker-compose.yml` dosyasının bulunduğu proje dizinine gidin ve şu komutu çalıştırın:

```bash
docker compose up -d --build
```

### 3. Panele Erişin

Konteyner başarıyla ayağa kalktığında web tarayıcınızı açın ve şu adrese gidin:
**http://localhost:8080** _(veya sunucunuzun_ip_adresi:8080)_

### 4. Hesap Oluşturun

Giriş sayfasındaki **"Create Account"** butonuna tıklayarak ilk kullanıcınızı oluşturun ve sitelerinizi eklemeye başlayın!

---

## 📁 Veri Kalıcılığı (Data Persistence)

Kullanıcı hesaplarınız, eklediğiniz monitörler ve geçmiş ping verileri güvenle saklanır. Bunun için yerel bir SQLite veritabanı kullanıyoruz.
Docker konteynerini silseniz veya yeniden başlatsanız bile, ana bilgisayarınızdaki `./data` klasörü silinmediği sürece **hiçbir veriniz kaybolmaz**.

_Not: Eğer projeyi başkasına sıfır, temiz bir şekilde vermek isterseniz, klasörü göndermeden önce `data` klasörünün içini tamamen boşaltmanız yeterlidir._

## ⚙️ Servisi Durdurmak

İzleme servisini tamamen durdurmak ve kapatmak için terminalde şu komutu çalıştırmanız yeterlidir:

```bash
docker compose down
```
