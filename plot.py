import pandas
import random
import numpy as np
import matplotlib.pyplot as plt
import math

def f1(t):
    return t * t

def f2(t):
    return t * np.log2(t)

t = np.arange(0.0, 10, 1)

plt.figure()
# plt.plot(t, f1(t), 'g-', t, f1(t), 'k')
plt.plot(t, f2(t), 'r-')
plt.show()
