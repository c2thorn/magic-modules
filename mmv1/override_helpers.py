
# move field to the bottom of the YAML by deleting it and re-appending
def move_down(field, parent_yaml):
    if field in parent_yaml and type(parent_yaml[field]) is not None:
        temp_item = parent_yaml[field]
        del parent_yaml[field]
        parent_yaml[field] = temp_item
    return parent_yaml

# set up recursive_search
def field_override_func(resource_yamls, resource_name, array_key, field_override, field_parts, field_overrides_key):
    resource_yamls[resource_name][array_key] = rescursive_search(field_parts, resource_yamls[resource_name][array_key], field_override, field_overrides_key)

# recursively search for nested fields in yaml given an array of parents
def rescursive_search(current_field_parts, parent_field_yaml, field_override, field_overrides_key):
    if len(current_field_parts) == 0:
        return parent_field_yaml
    # print(current_field_parts)
    current_field = current_field_parts[0]
    del current_field_parts[0]
    for i, field_yaml in enumerate(parent_field_yaml):
        if field_yaml['name'] == current_field:
            if len(current_field_parts) > 0:
                if 'item_type' in parent_field_yaml[i] and type(parent_field_yaml[i]['item_type']) is not None:
                    parent_field_yaml[i]['item_type']['properties'] = rescursive_search(current_field_parts, parent_field_yaml[i]['item_type']['properties'], field_override, field_overrides_key)
                elif 'value_type' in parent_field_yaml[i] and type(parent_field_yaml[i]['value_type']) is not None:
                    parent_field_yaml[i]['value_type']['properties'] = rescursive_search(current_field_parts, parent_field_yaml[i]['value_type']['properties'], field_override, field_overrides_key)
                else:
                    parent_field_yaml[i]['properties'] = rescursive_search(current_field_parts, parent_field_yaml[i]['properties'], field_override, field_overrides_key)
            else:
                parent_field_yaml[i][field_overrides_key] = field_override
                parent_field_yaml[i] = move_down('item_type', parent_field_yaml[i])
                parent_field_yaml[i] = move_down('properties', parent_field_yaml[i])
                return  parent_field_yaml
    return parent_field_yaml