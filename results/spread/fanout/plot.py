import pandas
import random
import matplotlib.pyplot as plt

def prepare():
    plt.subplots(1, 1, constrained_layout = True, figsize=(7,5))

def show():
    plt.xlim(left=0, right=1200)
    plt.ylim(top=21, bottom=0)
    plt.xlabel('Число экзмепляров')
    plt.ylabel('Сетевая нагрузка')
    plt.legend()
    plt.grid()
    plt.show()

data = pandas.read_csv('bench-merged.csv')
bounds = pandas.read_csv('log-bases-merged.csv')


# max
# L1 max
prepare()
mult1 = 1
mult2 = 1.5
plt.plot(data['n'], data['L1'], label='ML1')
plt.plot(bounds['n'], mult1*bounds['log2(n)'], label='log2(n)')
plt.plot(bounds['n'], mult2*bounds['log2(n)'], label=str(mult2)+'log2(n)')
show()

# L2 max
prepare()
mult1 = 1
mult2 = 1.7
plt.plot(data['n'], data['L2'], label='ML2')
plt.plot(bounds['n'], mult1*bounds['log2(n)'], label='log2(n)')
plt.plot(bounds['n'], mult2*bounds['log2(n)'], label=str(mult2)+'log2(n)')
show()

# L3 max
prepare()
mult1 = 1
mult2 = 2
plt.plot(data['n'], data['L3'], label='ML3')
plt.plot(bounds['n'], mult1*bounds['log2(n)'], label='log2(n)')
plt.plot(bounds['n'], mult2*bounds['log2(n)'], label=str(mult2)+'log2(n)')
show()
