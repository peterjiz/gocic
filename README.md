# GoCIC - a Citizenship & Immigration Canada (CIC) Automation Tool
Automatically retrieves your Citizenship &amp; Immigration Canada (CIC)'s Citizenship File, checks for updates, and sends you an email if there was any change noticed. 


GoCIC does 2 things: 

-It runs a Tick job that will automatically check for updates on your file every 15 mins and send you an email if changes were detected.

-It runs as a server on port 80 and listens for update requests at www.example.com/refresh or localhost/refresh and informs all applicants of their current statuses via email.


How to use: 
Enter your email account settings in mailserver.json 

Enter as many CIC applications you would like checked, and the corresponding emails to notify.


A working example cannot be provided for security reasons!


Feel free to do whatever you want with the code!


# Thanks 
Parts of sjakub' script were forked from http://bit.ly/1mJnr0e or http://pastebin.com/yDU0P5Z4


# Legal Disclaimer 
The author is in no way responsible for any illegal use of this software. I am also not responsible for any damages or mishaps that may happen in the course of using this software. Use at your own risk.
