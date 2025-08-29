# Josh's Unremarkable Mail Server
A mail server hobby project. (Almost) Implements SMTP and will eventually support IMAP for mail retreival. This project is currently very WIP (example: there's currently no way to retrieve mail from the server) so I don't recommend using it for anything other than testing at the moment.

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## Installation
The following instructions are only for installing/configuring JUMS. Acquiring a domain name and tls certs and properly configuring your DNS for email is outside the scope of this document.
1. Clone the repository:
```bash
git clone https://github.com/Queueue0/jums.git
```
2. Build the server:
```bash
go build ./cmd/server
```
3. Create a `config.toml`. The server expects this to be located in `$XDG_CONFIG_HOME/jums`, which by default will be `~/.config/jums`. You'll need to set the following values:
```toml
Domain = "<your domain as it appears in email addresses>"
Mxdomain = "<your server's domain as it appears in MX records>"
BoxesDir = "/path/to/your/mailboxes"
CertFile = "/path/to/your/tls/cert"
KeyFile = "/path/to/your/tls/key"
```
4. At this point you should be able to run the server (must run as root):
```bash
./server
```

## Usage
Configure your MUA of choice to connect to the server. You can use either TLS/SSL on port 465 or STARTTLS on port 587.

## Contributing
1. Fork the repository
2. Create a new branch: `git checkout -b feature-name`
3. Make your changes
4. Push your branch: `git push origin feature-name`
5. Create a pull request

## License
See [LICENSE.md](LICENSE.md) for full details
```
Copyright (C) 2025  Joshua Coffey

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
