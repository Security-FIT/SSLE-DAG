from scipy.stats import lognorm, expon, norm, kstest
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

# Load the data
df = pd.read_csv('node_degree.csv')
data = df["probabilities"]

# Fit a lognormal distribution
shape, loc, scale = lognorm.fit(data)

# Plot the histogram of the data
# count, bins, _ = plt.hist(data, bins=20, density=True, alpha=0.6, color='g', label="Histogram")

# Test the fit
kst_stat, p_value = kstest(data, 'lognorm', args=(shape, loc, scale))
print(f"KS Statistic: {kst_stat}, P-value: {p_value}")

# Generate random data from the lognormal distribution
new_data = lognorm.rvs(shape, loc, scale, size=20)

new_data = new_data * 56 + 4
rounded_data = [round(x) for x in new_data]
print("Max:", max(rounded_data))
print("Mean:", sum(rounded_data) / len(rounded_data))

plt.hist(rounded_data, bins=50, density=True, label="Histogram")

# # Lognormal PDF
# x = np.linspace(min(data), max(data), 1000)
# pdf_lognorm = lognorm.pdf(x, shape, loc, scale)
# plt.plot(x, pdf_lognorm, 'r-', label='Lognormal fit')

plt.legend()
plt.show()
