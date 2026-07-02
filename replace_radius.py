import os
import re
import glob

# Mapping for single-value border-radius replacements
# Key: regex pattern to match, Value: replacement string
replacements = [
    # 3px → radius-sm (4px) - small elements
    (r'border-radius:\s*3px(?![0-9])', 'border-radius: var(--radius-sm)'),
    # 4px → radius-sm (4px) - small elements
    (r'border-radius:\s*4px(?![0-9])', 'border-radius: var(--radius-sm)'),
    # 6px → radius-md (6px) - inputs, cards
    (r'border-radius:\s*6px(?![0-9])', 'border-radius: var(--radius-md)'),
    # 8px → radius-lg (12px) - cards (closest variable)
    (r'border-radius:\s*8px(?![0-9])', 'border-radius: var(--radius-lg)'),
    # 10px → radius-lg (12px) - badges, pills
    (r'border-radius:\s*10px(?![0-9])', 'border-radius: var(--radius-lg)'),
    # 12px → radius-lg (12px) - large cards
    (r'border-radius:\s*12px(?![0-9])', 'border-radius: var(--radius-lg)'),
    # 16px → radius-lg (12px) - panels/modals (use largest available)
    (r'border-radius:\s*16px(?![0-9])', 'border-radius: var(--radius-lg)'),
]

# Track changes
changes = []
total_files = 0
total_replacements = 0

# Find all .vue and .css files in frontend/src
files = []
for ext in ['*.vue', '*.css']:
    files.extend(glob.glob(f'frontend/src/**/{ext}', recursive=True))

for filepath in files:
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original = content
    file_changes = 0
    
    for pattern, replacement in replacements:
        new_content, count = re.subn(pattern, replacement, content)
        if count > 0:
            content = new_content
            file_changes += count
            total_replacements += count
    
    if file_changes > 0:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        total_files += 1
        changes.append(f'{filepath}: {file_changes} replacements')

print(f'Total files modified: {total_files}')
print(f'Total replacements: {total_replacements}')
print()
for change in changes:
    print(change)
