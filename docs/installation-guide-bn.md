# Hostvra ইনস্টলেশন ও ডেপ্লয়মেন্ট গাইডলাইন (বাংলা)

এই ডকুমেন্টটিতে ফিজিক্যাল ডেডিকেটেড সার্ভার এবং ভার্চুয়াল প্রাইভেট সার্ভারে (VPS) **Hostvra সার্ভার কন্ট্রোল প্যানেল** ইনস্টলেশন, কনফিগারেশন এবং রক্ষণাবেক্ষণের সম্পূর্ণ প্রোডাকশন নির্দেশিকা বর্ণিত হয়েছে।

---

## ১. সিস্টেমের প্রয়োজনীয়তা (System Requirements)

### সমর্থিত অপারেটিং সিস্টেম (Supported OS)
- **Ubuntu:** 22.04 LTS, 24.04 LTS (সবচেয়ে বেশি রেকমেন্ডেড)
- **Debian:** 11, 12
- **AlmaLinux / Rocky Linux / RHEL:** 9.x

### হার্ডওয়্যার স্পেসিফিকেশন
| রিসোর্স | ন্যূনতম (Minimum) | রেকমেন্ডেড (Production) | হাই ট্রাফিক ফ্লিট |
|---|---|---|---|
| **CPU** | ১ কোর (x86_64 বা ARM64) | ২ কোর | ৪+ কোর |
| **RAM** | ১ জিবি | ২ জিবি – ৪ জিবি | ৮ জিবি+ |
| **Storage** | ১০ জিবি SSD | ৪০ জিবি+ NVMe | ১০০ জিবি+ NVMe |
| **Network** | ১০০ Mbps, ১টি স্ট্যাটিক IPv4 | ১ Gbps, স্ট্যাটিক IPv4 + IPv6 | ১ Gbps+, ডেডিকেটেড আইপি |

---

## ২. ইনস্টলেশন পদ্ধতি

### পদ্ধতি ক: অটোমেটিক ওয়ান-লাইন ইনস্টল (GitHub থেকে সরাসরি - রেকমেন্ডেড)

আপনার লিনাক্স সার্ভারে (Ubuntu/Debian) রুট প্রিভিলেজ সহ সরাসরি GitHub রিপোজিটরি থেকে নিচের কমান্ডটি রান করুন:

```bash
curl -fsSL https://raw.githubusercontent.com/edge-tec/Hostvra/main/deployment/installer/install.sh | sudo bash
```
*(অথবা শর্টলিংক ডোমেইন কনফিগার করা থাকলে: `curl -fsSL https://install.hostvra.com | sudo bash`)*

### পদ্ধতি খ: গিট রিপোজিটরি ক্লোন করে ইনস্টল

```bash
git clone https://github.com/edge-tec/Hostvra.git
cd Hostvra
sudo bash deployment/installer/install.sh
```

### পদ্ধতি গ: লোকাল ডেভেলপমেন্ট ও ডেমো টেস্ট (Local Workstation)

আপনার পার্সোনাল কম্পিউটার বা ল্যাপটপে পরীক্ষা করতে:

```bash
git clone https://github.com/edge-tec/Hostvra.git
cd Hostvra

# টার্মিনাল ১: ব্যাকএন্ড API সার্ভার চালু (http://localhost:8080)
cd apps/api
go run ./cmd/server

# টার্মিনাল ২: ফ্রন্টএন্ড ওয়েব কন্ট্রোল প্যানেল চালু (http://localhost:3000)
cd apps/web
npm install
npm run dev
```

ইনস্টলার যা যা নিজে কনফিগার করে:
1. OS, আর্কিটেকচার এবং মেমোরি স্বয়ংক্রিয় যাচাই।
2. Nginx, Postfix, Dovecot, Rspamd, CA সার্টিফিকেট ইনস্টলেশন।
3. নন-রুট আনপ্রিভিলেজড সিস্টেম ইউজার `hostvra` এবং মেইল ইউজার `vmail` (UID 5000) তৈরি।
4. ক্রিপ্টোগ্রাফিক JWT সিক্রেট কি এবং প্রাথমিক অ্যাডমিন পাসওয়ার্ড জেনারেশন।
5. Systemd ব্যাকগ্রাউন্ড সার্ভিস রেজিস্ট্রেশন (`hostvra-api`, `hostvra-agent`)।
6. ফায়ারওয়ালের পোর্ট স্বয়ংক্রিয়ভাবে ওপেন ও সিকিউর করা।

ইনস্টলেশন শেষে টার্মিনালে লগইন লিংক এবং অ্যাক্সেস ক্রেডেনশিয়াল প্রদর্শিত হবে:
```text
======================================================================
             🎉 HOSTVRA INSTALLATION COMPLETED SUCCESSFULLY!           
======================================================================
  • Panel URL:      http://<SERVER_IP>:8080
  • Admin Email:    admin@hostvra.local
  • Password:       <সিকিউর_অটো_পাসওয়ার্ড>
======================================================================
```

---

## ৩. ম্যানুয়াল ইনস্টলেশন পদ্ধতি (Manual Enterprise Setup)

যেসকল রেস্ট্রিক্টেড বা এয়ার-গ্যাপড সার্ভারে রিমোট স্ক্রিপ্ট রান করার অনুমতি নেই:

### ধাপ ১: অপারেটিং সিস্টেম প্যাকেজসমূহ ইনস্টল
```bash
# Ubuntu / Debian
sudo apt-get update -qq
sudo apt-get install -y curl wget tar gzip openssl ufw nginx ca-certificates \
    postfix dovecot-imapd dovecot-pop3d dovecot-lmtpd rspamd
```

### ধাপ ২: সিস্টেম ইউজার ও ডিরেক্টরি সেটআপ
```bash
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/hostvra -m hostvra 2>/dev/null || true
sudo groupadd -g 5000 vmail 2>/dev/null || true
sudo useradd -r -u 5000 -g vmail -s /usr/sbin/nologin -d /var/mail/vhosts -m vmail 2>/dev/null || true

sudo mkdir -p /etc/hostvra /var/lib/hostvra/backups /var/lib/hostvra/www /var/log/hostvra /var/mail/vhosts
sudo chown -R hostvra:hostvra /var/lib/hostvra /var/log/hostvra
sudo chown -R vmail:vmail /var/mail/vhosts
sudo chmod 750 /var/lib/hostvra /var/log/hostvra
sudo chmod 770 /var/mail/vhosts
sudo chmod 700 /etc/hostvra
```

### ধাপ ৩: এনভায়রনমেন্ট কনফিগারেশন (`/etc/hostvra/api.env`)
```bash
JWT_SECRET=$(openssl rand -hex 32)

sudo bash -c "cat > /etc/hostvra/api.env" << EOF
PORT=8080
HOST=0.0.0.0
JWT_SECRET=${JWT_SECRET}
LOG_FORMAT=json
DATA_DIR=/var/lib/hostvra
EOF

sudo chmod 600 /etc/hostvra/api.env
sudo chown hostvra:hostvra /etc/hostvra/api.env
```

### ধাপ ৪: বাইনারিসমূহ স্থাপন
```bash
sudo cp bin/hostvra-api /usr/local/bin/hostvra-api
sudo cp bin/hostvra-agent /usr/local/bin/hostvra-agent
sudo cp bin/hostvra /usr/local/bin/hostvra
sudo chmod +x /usr/local/bin/hostvra-api /usr/local/bin/hostvra-agent /usr/local/bin/hostvra
sudo ln -sf /usr/local/bin/hostvra /usr/local/bin/hostvra-update
```

### ধাপ ৫: Systemd সার্ভিস সক্রিয়করণ
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now hostvra-api.service
sudo systemctl enable --now hostvra-agent.service
```

---

## ৪. ফায়ারওয়াল পোর্ট তালিকা (Firewall Port Reference)

| পোর্ট | প্রোটোকল | বিবরণ |
|---|---|---|
| `22` | TCP | SSH রিমোট অ্যাক্সেস |
| `8080` | TCP | Hostvra কন্ট্রোল প্যানেল ওয়েব UI |
| `80` | TCP | HTTP ওয়েব ট্র্যাফিক |
| `443` | TCP | HTTPS সিকিউর ওয়েব ট্র্যাফিক ও SSL |
| `25` | TCP | SMTP মেইল সার্ভার রিলে |
| `465` | TCP | SMTPS সুরক্ষিত মেইল পাঠানো |
| `587` | TCP | SMTP সাবমিশন (STARTTLS) |
| `993` | TCP | IMAPS সুরক্ষিত ইমেইল রিডিং |
| `995` | TCP | POP3S সুরক্ষিত ইমেইল ডাউনলোড |

---

## ৫. অ্যাডমিন লগইন ও প্রাথমিক সেটআপ চেকলিস্ট

### ডিফল্ট লগইন ক্রেডেনশিয়ালস

| পরিবেশ (Environment) | প্যানেল URL | অ্যাডমিন ইমেইল | পাসওয়ার্ড | রোল |
|---|---|---|---|---|
| **লোকাল টেস্ট / ডেমো** | `http://localhost:3000/login` | `admin@hostvra.com` | `SuperSecretP@ss123!` | Super Admin / Owner |
| **প্রোডাকশন সার্ভার** | `http://<SERVER_IP>:8080` | `admin@hostvra.local` | ইনস্টলেশন সামারিতে জেনারেট হওয়া পাসওয়ার্ড | Super Admin / Owner |

> 💡 **ডেমো টিপস**: ব্রাউজারের লগইন পেজে সরাসরি **"Prefill Demo Credentials"** বাটনে ক্লিক করলে `admin@hostvra.com` এবং `SuperSecretP@ss123!` অটো-ফিল হয়ে যাবে।

### সার্ভার থেকে পাসওয়ার্ড দেখা
প্রোডাকশন সার্ভারের এনভায়রনমেন্ট কনফিগারেশন দেখতে:
```bash
sudo cat /etc/hostvra/api.env
```

### নতুন অ্যাডমিন ইউজার রেজিস্টার করার কমান্ড
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@yourdomain.com",
    "password": "YourStrongPassword123!",
    "full_name": "System Administrator",
    "organization_name": "Hostvra Enterprise"
  }'
```

### প্রাথমিক সিকিউরিটি চেকলিস্ট
1. ব্রাউজারে `http://<SERVER_IP>:8080` (বা লোকাল মেশিনে `http://localhost:3000`) এ ঢুকে অ্যাডমিন ক্রেডেনশিয়ালস দিয়ে লগইন করুন।
2. **Settings > Profile** থেকে এডমিন পাসওয়ার্ড পরিবর্তন করে নিজস্ব শক্তিশালী পাসওয়ার্ড দিন।
3. **SSL Certificates** মেনুতে গিয়ে কন্ট্রোল প্যানেলের জন্য ফ্রি Let's Encrypt SSL ইস্যু করুন।
4. **Web Servers** মেনুতে গিয়ে প্রয়োজন অনুসারে **Nginx**, **Apache**, **OpenLiteSpeed** বা **LiteSpeed Enterprise** এক্টিভেট করুন।
5. **PHP Management** থেকে প্রয়োজনীয় PHP সংস্করণ (8.4, 8.3, 8.2, 8.1) এবং এক্সটেনশন ইনস্টল করুন।
6. **Backups > Storage Providers** এ গিয়ে S3 / Cloudflare R2 কানেক্ট করে অফ-সাইট ব্যাকআপ নিশ্চিত করুন।

---

## ৬. লাইভ আপডেট ও ইনস্ট্যান্ট রোলব্যাক

Hostvra-তে কোনো গ্রাহকের ওয়েবসাইট বন্ধ না করে নিরাপদে ব্যাকগ্রাউন্ডে আপডেট করা যায়:

```bash
# আপডেট চেক
hostvra update check

# টার্মিনাল স্পিনার সহ লাইভ আপডেট
sudo hostvra update install -y

# কোনো সমস্যা হলে তৎক্ষণাৎ পূর্বের সংস্করণে নিরাপদ রোলব্যাক
sudo hostvra update rollback -y
```

সার্ভারে ইনস্টলার স্ক্রিপ্টটি পুনরায় রান করলে (`sudo bash install.sh`), ইনস্টলার বিদ্যমান ডেটা বা ডেটাবেস অক্ষুণ্ণ রেখে স্বয়ংক্রিয়ভাবে **Safe Upgrade Mode** এ আপডেট সম্পন্ন করে।
