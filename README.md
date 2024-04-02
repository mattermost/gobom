# gobom

An extensible [CycloneDX](https://cyclonedx.org/) BOM generator and [Dependency-Track](https://dependencytrack.org/) API client written in Go.

## Installation

```
go install github.com/mattermost/gobom/cmd/gobom@latest
```

## Usage

```
gobom generate --recurse --url https://dependency-track.example.com --key $DEPENDENCY_TRACK_API_KEY --project projectname@version /path/to/project/
```

### What does it do?

`gobom generate` generates a CycloneDX BOM for your software project. Dependencies from multiple ecosystems can be included in a single BOM file: e.g. a React Native project can be scanned recurisvely using the default generators, and the output will include both JavaScript and native dependencies.

Generated component information includes valid PURLs for all components (compatible with [OSS Index](https://ossindex.sonatype.org/)) and descriptions that show the ecosystem the dependency was included through, as well as the shortest path to the root project for transitive dependencies.

`gobom upload` uploads an existing BOM file to a Dependency-Track server for analysis. Upload can also be invoked through `generate` simply by including the relevant flags.

### What doesn't it do?

`gobom` does not analyze vulnerabilities; that's what the Dependency-Track integration is for. It only aims to generate an accurate listing of the dependencies in a project.

`gobom` only supports a very limited subset of the CycloneDX specification required for successful vulnerability analysis. In particular, it does not generate full dependency graphs but only component listings. This is because Dependency-Track currently has no support for displaying dependency graphs; generating them would be of no benefit to Dependency-Track users.

`gobom` has only been tested for interoperability with Dependency-Track. Interoperability with any other CycloneDX tooling is not guaranteed or even expected at this time.

## Supported generators

Built-in BOM generators include support for:

 - Go modules (`generators/gomod`)
 - npm (`generators/npm`)
 - CocoaPods (`generators/cocoapods`)
 - Gradle (`generators/gradle`)

The npm and CocaPods generators are based on parsing lockfiles and have no runtime dependencies. The Go and Gradle generators respectively depend on the `go` and `gradle` command line tools at runtime. Gradle wrappers are supported.

Help specific to each generator can be viewed using `gobom help generators/gradle`; just replace the generator name with the one that interests you.

## Adding custom generators

`gobom` was designed to be extensible. Have a legacy project where you're tracking dependencies in a custom text file? Want to add support for another language but not ready to contribute to the main project just yet? No problem.

`gobom` can be extended without forking the main project: just implement your own generator, import it and `github.com/mattermost/gobom/commands` in your main package, and call `commands.Execute()` to get the full command line interface.

See the example [here](./examples/custom_generator) for more details.
