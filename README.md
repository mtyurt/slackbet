##Simple bet integration for slack
This hobby project is used in my workplace. In this simple bet system, each employee tries to guess new user count of the month and half of the office buys coffee to the other half of the office at the end of the month. 

This project needs to be run in a server continuously. There should be a bot user in Slack and proper configuration should be assigned. Our use-case runs on a combination of command line integration and bot user integration in Slack.

This project depends on a Redis instance and I just figured that the port is hard-coded. I'll make it more configurable later.

####Future imporvements
- Make this readme more meaningful and state all features
- Tests run on Redis now, they should run on a mock repository layer
- Consider usage of a simpler persistence, like yaml
- Make project more configurable
- The related number regarding the bet result is post to a channel in Slack. With the help of an awesome regex find out the number and update winnerScore of the bet over a chat-bot
- Instead of /bet command usage, use chat bot's dm support to save bets
