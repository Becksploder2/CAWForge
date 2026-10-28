# CAWForge

Windows test-build automation is configured in [GitHub Actions](https://github.com/Becksploder2/CAWForge/actions/workflows/windows-test.yml).

**Build prerequisites:** This repository currently needs the full v0.9.14 application source (`main.go`), the `assets/` directory with `assets/templates/Female/F1.png` and male templates, and companion files before the workflow can build a usable ZIP.

Once source and assets are uploaded, run **Actions → CAWForge Windows Test Build → Run workflow**. Download the **CAWForge-Windows-Test** artifact and extract its complete contents. This is a diagnostic build, not a stable release.

The Windows UI and face-texture generation changes are not yet verified on a Windows desktop.
