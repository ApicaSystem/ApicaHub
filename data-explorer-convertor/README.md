# Data Explorer Convertor

This project converts public grafana JSON dashboard files into a format that Data Explorer supports.

## Build the Application

To build the application, run:

```bash
go build .
```

## Run the Application
To build the application, run:

```
./data-explorer-convertor <input-file-path> <output-file-path> <data-explorer-name>
```

```<input-file-path>```: Path to the input data file.<br />
```<output-file-path>:``` Path for the converted output file.<br />
```<data-explorer-name>:``` Name of the Data Explorer for conversion.