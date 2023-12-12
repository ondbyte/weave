## weave cli

find plugin implementations inside `./plugins` folder

first build the plugin by `cd`ing to a plugin directory, for example from the root of the project
```
cd ./plugins/plugin_a
```
and build it
```
go build .
```

now run the cli (dont forget to change the directory to cmd subfolder under project folder)
```
cd ../../cmd
```
and

```
go run . --path ./test_hcl_files --document "test-document"
```

### FOR HELP
```
go run -h
```