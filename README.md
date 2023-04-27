PostSuggesterBot
==========================
_(for a Telegram channel)_

[![CI Build](https://github.com/kozalosev/PostSuggesterBot/actions/workflows/ci-build.yml/badge.svg?branch=main&event=push)](https://github.com/kozalosev/PostSuggesterBot/actions/workflows/ci-build.yml)

Some user suggests a message for the channel:

![image](https://user-images.githubusercontent.com/25857981/234779419-6b2ef462-bdc1-40b6-a3b1-896051740ff5.png)

The admins of the channel can approve and publish it:

![image](https://user-images.githubusercontent.com/25857981/234779194-9dde2ea0-68e6-4d3d-a411-a4191755f609.png)

The count of required approvals is configurable.


Architecture
------------
The bot consists of three components running within Docker containers:
1) the main application written in Go;
2) PostgreSQL as the main datastore;
3) Redis as a storage for temporary data.

The application is based on the [goSadTgBot][goSadTgBot] framework and consists of:
* [DTOs](db/dto) and [repositories](db/repo) to work with database entities;
* [Handlers](handlers) for processing of messages and callback queries.

[goSadTgBot]: https://github.com/kozalosev/goSadTgBot/

Commands
--------
For usual users:
* `/help` and `/start` to get help;
* `/language` and `/lang` to change the language.

For admins:
* `/promote` authors and other admins.

Required parameters
-------------------
Set the following environment variables in your *.env* file:
* **API_TOKEN** — the token of your bot issued by [@BotFather](https://t.me/BotFather);
* **CHANNEL_ID** — the identifier of your channel ([how to find](https://www.alphr.com/find-chat-id-telegram/));
* **CHANNEL_NAME** — the name of the channel which is used in the help text;
* **ADMIN_CHAT_ID** — the identifier of your group chat with other administrators responsible for approvals.
* **REQUIRED_APPROVALS** — a count of required approvals; the value `1` is used as a fallback.

Used by
-------
* [WellOfDesires](https://t.me/WellOfDesires)
