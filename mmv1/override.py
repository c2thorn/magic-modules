import os, pathlib, sys, io
import ruamel.yaml, my_ez_yaml, override_helpers


yaml = ruamel.yaml.YAML()
yaml.preserve_quotes = True
yaml.indent(mapping=2, sequence=4, offset=2)

# folders = [ f.path for f in os.scandir("products/") if (f.is_dir() and f.name.startswith('a'))]
folders = ["products/accessapproval", "products/activedirectory"]

print(folders)

# merges terraform.yaml into 
for folder in folders:
    resource_yamls = {}
    override_yaml = None
    override_filename = ""

    for filename in os.listdir(folder):
        print(filename)
        override_file = False
        product_file = False
        f = os.path.join(folder,filename)
        if os.path.isfile(f):
            suffix = pathlib.Path(f).suffix
            if suffix != '.yaml':
                print("skipping ", f)
                continue
            match filename:
                case 'terraform.yaml':
                    override_file = True
                    override_filename=f

                    # remove problematic text that prevents YAML parsing
                    with open(f, "r") as input:
                        with open("temp.yaml", "w") as output:
                            for line in input:
                                if "# This is for copying files over" in line or "files: !ruby/object:Provider::Config::Files" in line:
                                    break
                                output.write(line)
                        os.replace('temp.yaml', f)
                case 'product.yaml':
                    # product overrides rare enough to be handled manually
                    continue
            with open(f, "r") as file:
                loaded_yaml = yaml.load(file)
                if override_file:
                    override_yaml = loaded_yaml
                else:
                    resource_yamls[loaded_yaml['name']] = loaded_yaml
    if override_filename == "":
        print ("no terraform.yaml skipping", folder)
        continue

    # Merge override YAML into primary resource YAML in-memory
    for resource_name in override_yaml['overrides']:
        resource_yaml = resource_yamls[resource_name]
        for override_key in override_yaml['overrides'][resource_name]:
            override = override_yaml['overrides'][resource_name][override_key]
            if override_key == 'properties':
                for field in override_yaml['overrides'][resource_name]['properties']:
                    for field_overrides_key in override_yaml['overrides'][resource_name]['properties'][field]:
                        field_override = override_yaml['overrides'][resource_name]['properties'][field][field_overrides_key]
                        override_helpers.field_override_func(resource_yamls, resource_name, 'properties', field_override, field.split("."), field_overrides_key)
                        if 'parameters' in resource_yamls[resource_name] and type(resource_yamls[resource_name]['parameters']) is not None:
                            override_helpers.field_override_func(resource_yamls, resource_name, 'parameters', field_override, field.split("."), field_overrides_key)
            else:
                resource_yamls[resource_name][override_key] = override
        resource_yamls[resource_name] = override_helpers.move_down('parameters', resource_yamls[resource_name])
        resource_yamls[resource_name] = override_helpers.move_down('properties', resource_yamls[resource_name])

    # rewrite resource yaml's
    prefix = "# Copyright 2023 Google Inc.\n# Licensed under the Apache License, Version 2.0 (the \"License\");\n# you may not use this file except in compliance with the License.\n# You may obtain a copy of the License at\n#\n#     http://www.apache.org/licenses/LICENSE-2.0\n#\n# Unless required by applicable law or agreed to in writing, software\n# distributed under the License is distributed on an \"AS IS\" BASIS,\n# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n# See the License for the specific language governing permissions and\n# limitations under the License.\n\n--- !ruby/object:Api::Resource\n"
    for resource_key in resource_yamls:
        filename = resource_key+'.yaml'
        filename = os.path.join(folder,filename)
        f = open(filename, "w")
        f.write(prefix)
        for line in my_ez_yaml.to_string(resource_yamls[resource_key]).splitlines():
            if line == "%YAML 1.2" or line == "--- !ruby/object:Api::Resource":
                continue
            f.write(""+line+"\n")

    # delete terraform.yaml
    os.remove(override_filename)


