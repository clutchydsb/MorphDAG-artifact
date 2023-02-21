## Introduction

The artifact includes the source code of all key modules of MorphDAG mentioned in the paper. Specifically, the artifact contains the following sub-directories:

- `./Prototype`, which includes the source code of MorphDAG implementation.
- `./Baselines`, which includes the source code of baseline approaches, i.e., OHIE and Occam, as well as Nezha.
- `./Workloads`, which includes the workload tool that contains Ethereum transaction data used in experiments.
- `./Experiments`, which includes all the experimental scripts used to help deploy systems and run experiments.

Besides, the `README.md` file in each sub-directory provides deployment and usage instructions for users.

## Dependencies

The Artifact has the following requirements.

**Hardware Requirement:** 

The distributed node environment contains 50 machines, each of which is configured by:

- CPU: Intel Xeon Platinum 8369B @3.5GHz, 16 cores
- DRAM: 32GB
- Disk: 100GB SSD

In the local node environment, you can configure the number of processes running on the local based on the number of MorphDAG nodes. It is recommended that the number of running processes is less than 100 nodes If the configuration of the local machine is the same as above.

**Software Requirement:** 

Operating system: Ubuntu 20.04 LTS (other latest releases may also be OK)

Runtime language: Golang v1.18 (All software dependencies are included in the go.mod file)

*Notice: we have built all dependencies with Golang v1.20, which is the latest version of Golang until now.*

## Usage

1. Please refer to README files in `./Prototype` and .`/Baselines` to learn how to build MorphDAG and other baseline systems. 
2. Please refer to the README file in `./Workloads` for obtaining a workload dataset file for experimental evaluations. 
3. Please refer to the README file in `./Experiments` to learn how to run nodes in a local/distributed node environment to reproduce experimental results.