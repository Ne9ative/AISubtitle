# Binaires embarqués (non versionnés)

Ce dossier doit contenir, **avant le build**, les exécutables MKVToolNix :

- `mkvmerge.exe`
- `mkvextract.exe`

Ils sont embarqués dans l'application via `go:embed` (voir `../embed.go`) puis
auto-extraits dans `%LOCALAPPDATA%\AISubtitlePro\bin\` au 1er lancement.

Ils ne sont **pas versionnés** (voir `.gitignore`, règle `*.exe`). Pour préparer
un build, copiez-les depuis une installation MKVToolNix
(https://mkvtoolnix.download/) dans ce dossier.

`llama-server.exe` (CUDA) n'est **pas** embarqué : il est téléchargé
automatiquement au 1er lancement (voir `../download.go`).
