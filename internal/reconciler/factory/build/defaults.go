package build

// This file is the bridge between the config.* default structs and the build.*Opts
// structs that each per-component factory passes to NewContainer / NewService / etc.
// Per-component coalescers (e.g. ApplyEventsReaderDefaults) will live in their own
// files inside build/<component>/ — those packages depend on this package and on
// internal/reconciler/config. Stage 0 leaves this file empty so the package compiles;
// downstream stages will add helpers like:
//
//   func MergeContainerDefaults(opts ContainerOpts, def config.FluentbitDefaults) ContainerOpts
//
// only as concrete needs surface.
