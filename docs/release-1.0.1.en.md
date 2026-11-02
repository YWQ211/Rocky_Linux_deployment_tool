# Rockdev 1.0.1

[中文](release-1.0.1.md) · English · [日本語](release-1.0.1.ja.md)

On Rocky 8, system Python resolves through alternatives to `/usr/libexec/platform-python3.6`. Version 1.0.0 classified it as a custom runtime and refused explicit batch updates containing Python. Version 1.0.1 accepts this exact target for standard Python entrypoints. Other libexec programs, custom locations and isolated environments keep their existing protections.

Integration checks read the index-url parsed by pip's actual install command, supporting pip 9 as well as newer versions. Rejection cases require CLI exit code 1; missing commands and timeouts cannot pass. Any failed case makes the script exit nonzero.

Commands are unchanged. New artifacts go to `dist/1.0.1/`; 1.0.0 is preserved. See the [1.0 guide](release-1.0.en.md) and [retest results](test-results-1.0.1.en.md).
