# GoCIC - a Citizenship & Immigration Canada (CIC) Automation Tool
Automatically retrieves your Citizenship &amp; Immigration Canada (CIC)'s Citizenship File, checks for update, and sends you an email if there was any change noticed. 

Also developed to run as a server, listening on port 80. www.example.com/refresh or localhost/refresh will send you an email letting you know of the current status.

Mail Server & CIC Requests (multiple) can be configured in mailserver.json and cicapplications.json. 
A default example was provided.

Parts of the script were forked from: 
http://www.canadavisa.com/canada-immigration-discussion-board/ecas-fetching-script-t385049.0.html 
http://pastebin.com/yDU0P5Z4
