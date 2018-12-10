# Building DappleyV8 for Windows

## Depend Tools

### MingW64

Download [MingW64](https://sourceforge.net/projects/mingw-w64/).

Install X64 version, and add bin to PATH.

### Visual Studio 2017 (Needed by ChromeV8)

As of September, 2017 (R503915) Chromium requires Visual Studio 2017 (15.7.2) to build. The clang-cl compiler is used but Visual Studio's header files, libraries, and some tools are required. Visual Studio Community Edition should work if its license is appropriate for you. You must install the “Desktop development with C++” component and the “MFC and ATL support” sub-component. This can be done from the command line by passing these arguments to the Visual Studio installer that you download:

--add Microsoft.VisualStudio.Workload.NativeDesktop
    --add Microsoft.VisualStudio.Component.VC.ATLMFC --includeRecommended
You must have the version 10.0.17134 Windows 10 SDK installed. This can be installed separately or by checking the appropriate box in the Visual Studio Installer.

The SDK Debugging Tools must also be installed. If the Windows 10 SDK was installed via the Visual Studio installer, then they can be installed by going to: Control Panel → Programs → Programs and Features → Select the “Windows Software Development Kit” → Change → Change → Check “Debugging Tools For Windows” → Change. Or, you can download the standalone SDK installer and use it to install the Debugging Tools.

## Build chrome V8

If you want to build v8 by yourself, please reference [Build Windows Chrome](https://chromium.googlesource.com/chromium/src/+/master/docs/windows_build_instructions.md). A prebuild version of v8 is included in this project.

## Build DappleyV8

Open dappleyv8.sln and build x64 version.

The dll will copy to dapp running path automated.

## Run dapp

Must copy jslib from go-dappley\contract to go-dappley\dapp, because linux symbol link takes no effect in windows.

```
cd go-dappley\dapp
.\dapp -f conf\seed.conf
```

## Run Cli

First must copy dll files to dapp\cli.

Then run cli commands.

## Known issues

* The test cases of contract failed for unkonwn reason. 
