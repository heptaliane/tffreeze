# tffreeze

`tffreeze` is the tool to substitute variables in the HCL files.

# Usage
``` bash
tffreeze [OPTION]... [FILE]...
```

## Substitute HCL files with tfvars
You can substitute variables with `.tfvars` file.

``` bash
tffreeze -var-file <tfvars file> [FILE]...
```

### Example
When you run `tffreeze` with "main.tf" and "variable.tfvars", you can get
substituted file named "main.freeze.tf".

```
tffreeze -var-file variables.tfvars main.tf
```

#### main.tf
``` hcl
resource "google_compute_instance" "default" {
    name         = var.instance_name
    machine_type = "n2-standard-${var.cpus}"
    zone         = var.zone

    tags = var.tags

    boot_disk {
        initialize_params {
            image  = var.disk_image
            labels = var.disk_labels
        }
    }
}
```

#### variables.tf
``` hcl
instance_name = "default-instance"
cpus          = 2

tags = ["foo", "bar"]

labels = {
    baz = "qux"
}
```

#### main.freeze.tf
``` hcl
resource "google_compute_instance" "default" {
    name         = "default-instance"
    machine_type = "n2-standard-2"
    zone         = var.zone

    tags = ["foo", "bar"]

    boot_disk {
        initialize_params {
            image  = var.disk_image
            labels =  {
                baz = "qux"
            }
        }
    }
}
```
