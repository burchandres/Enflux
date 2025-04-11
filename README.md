# Enflux
Package Enflux is a Go package for building structured, DAG-based data analytics pipelines.
It introduces the concept of a RoutineGraph: a directed acyclic graph with a single root node and one or more terminal nodes.
Each node, called a Routine, encapsulates a discrete unit of processing logic.

Enflux is ideal for defining complex, modular workflows where data flows from a root through a network of connected routines.
It supports branching, fan-out, and fan-in patterns, making it well-suited for real-time analytics, ETL systems, and custom pipeline engines.

Features:
  - Directed acyclic graph (DAG) execution model
  - Single entrypoint (root node), multiple terminal nodes
  - Modular Routine nodes with customizable behavior
  - Support for parallel execution paths
  - Support for parallel execution on a Routine scale
  - Minimal dependencies and easy integration
