- => https://tedsummer.com
- => gemini://tedsummer.com
- => https://git.sr.ht/~macintoshpie/macintoshpie.srht.site
- hosting provided by sourcehut pages
  - see .build.yml
- dev server
  - generate certs
    - ```bash
./scripts/genDevCert.sh
```
  - run dev server
    - ```bash
go run ./cmd/runDev
```
  - edit ./me.txt and ./me.tmpl.html
  - visit the site at http://localhost:8080
- build
  - ```bash
go run ./cmd/build
```
