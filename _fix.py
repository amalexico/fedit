c = open('main.go', encoding='utf-8').read()
c = c.replace('flag.Int("line", 0,', 'flag.String("line", "",', 1)
c = c.replace('flag.Int("end", 0,', 'flag.String("end", "",', 1)
open('main.go', 'w', encoding='utf-8', newline='\n').write(c)