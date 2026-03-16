Output images/prtr-setup-doctor.gif

Require bash
Require git

Set Theme "TokyoNightStorm"
Set Shell "bash"
Set FontSize 24
Set Width 1400
Set Height 900
Set Padding 28
Set Framerate 18
Set TypingSpeed 35ms
Set WindowBar Colorful

Hide
Type "export PATH=/tmp/prtr-demo/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
Enter
Type "export XDG_CONFIG_HOME=/tmp/prtr-demo/config"
Enter
Type "export XDG_DATA_HOME=/tmp/prtr-demo/data"
Enter
Type "export PS1='$ '"
Enter
Type "cd /Users/koo/dev/translateCLI-brew"
Enter
Type "./scripts/demos/setup_demo_env.sh"
Enter
Sleep 1s
Type "cd /tmp/prtr-demo/workspace"
Enter
Show

Type "prtr go fix 'Find the real reason this test is failing.' --dry-run"
Enter
Sleep 5s
