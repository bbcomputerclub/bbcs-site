# BB-Community-Service
The proposed community service website. The current one can be found [here](http://www2.blindbrook.org/test/zkcs/login.html).

## Building
```bash
git clone https://github.com/bbcomputerclub/bbcs-site.git
cd bbcs-site
go build *.go
```

## `data/` directory
The data directory contains files that contain information about the students and their hours.

* `students.csv` (required) - a csv file. `Name,GradYear,Email,Late`
* `entries-XXXX.json` (where XXXX is the graduation year) - the entries for that grade
* `admins.txt` - a list of admins' emails
