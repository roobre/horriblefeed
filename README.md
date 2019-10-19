# ⬇️ Horriblefeed

Horriblefeed is a service which automatically adds new anime chapters from a list of feeds to transmission.
It will only download new chapters from series that are already in the trnasmission torrent list.

Multiple feeds can be configured via a config file, for which there is a sample called `horriblefeed_example.yml` located in the root folder of this repo. The program will search for a file named `horriblefeed.yml`* in one of the following paths:

* `.` (Current working directory)
* `$XDG_CONFIG_HOME/horriblefeed/`
* `$HOME/.config/horriblefeed/`

\* Note: Other formats are supported as well, as long as the semantic meaning is the same as the one provided in the example yaml.

Horriblefeed will only request the first page for feeds listed in the config file.
