from setuptools import setup, find_packages

# Read in the requirements.txt file
with open("requirements.txt") as f:
    requirements = f.read().splitlines()

setup(
    name="tourneypdf",
    version="0.1.0",
    author="César del Solar",
    author_email="delsolar@gmail.com",
    description="A PDF scoresheet generator for Woogles.io tournaments",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    url="http://woogles.io/",
    package_dir={"": "src"},
    packages=find_packages(where="src"),
    install_requires=requirements,
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.7",
)
