import ruamel.yaml
from io import StringIO
from pathlib import Path
import os

# setup loader (basically options)
yaml = ruamel.yaml.YAML()
yaml.version = (1, 2)
yaml.indent(mapping=2, sequence=4, offset=2)
yaml.allow_duplicate_keys = True
yaml.explicit_start = False
# show null
yaml.representer.add_representer(
    type(None),
    lambda self, data: self.represent_scalar(u'tag:yaml.org,2002:null', u'null')
)

def to_string(obj, options=None):
    if options == None: options = {}
    string_stream = StringIO()
    yaml.dump(obj, string_stream, **options)
    output_str = string_stream.getvalue()
    string_stream.close()
    return output_str