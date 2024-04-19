import matplotlib.pyplot as plt

goroutines = [10, 100, 1000]

processing_times = [4 * 60 + 36.0509241, 2.6589316, 4.0643626 ]



plt.figure(figsize=(10, 6))
plt.plot(goroutines, processing_times, marker='o', linestyle='-')
plt.xlabel('Number of Goroutines')
plt.ylabel('Processing Time (seconds)')
plt.grid(True)
plt.xticks(goroutines)
plt.show()