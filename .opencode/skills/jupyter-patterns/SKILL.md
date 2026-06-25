---
name: jupyter-patterns
description: "Apply best practices for Jupyter notebooks: cell ordering, reproducibility, parameterisation. Use when working with .ipynb files."
---

# SKILL: Jupyter Notebook Patterns

## Notebook Structure
Cells should follow this order:
1. **Setup** — imports, configuration, constants
2. **Data Loading** — load raw data with validation
3. **EDA** — exploratory data analysis, distributions, correlations
4. **Preprocessing** — cleaning, feature engineering
5. **Modeling** — model training and evaluation
6. **Visualization** — charts and figures
7. **Conclusions** — findings summary, next steps

## Parameterized Execution with Papermill
```python
# In notebook cell tagged with "parameters":
# Click: View -> Cell Toolbar -> Tags -> add "parameters" tag

dataset = "data/train.csv"  # papermill will override this
output_dir = "outputs"
n_estimators = 100
random_state = 42
```

```bash
# Execute with papermill
papermill input.ipynb output.ipynb \
  -p dataset "data/test.csv" \
  -p n_estimators 200 \
  -p random_state 0
```

## Programmatic Notebook Creation
```python
import nbformat as nbf

nb = nbf.v4.new_notebook()
nb.cells = [
    nbf.v4.new_markdown_cell("# Analysis: {Title}"),
    nbf.v4.new_code_cell("import pandas as pd\nimport numpy as np"),
    nbf.v4.new_code_cell("df = pd.read_csv('data.csv')\ndf.head()"),
]

with open('analysis.ipynb', 'w') as f:
    nbf.write(nb, f)
```

## Export
```bash
# To HTML (with outputs)
jupyter nbconvert --to html --execute notebook.ipynb

# To PDF
jupyter nbconvert --to pdf --execute notebook.ipynb

# Execute in place
jupyter nbconvert --to notebook --execute --inplace notebook.ipynb
```

## Git Hygiene
```bash
# Clear outputs before commit
jupyter nbconvert --to notebook --ClearOutputPreprocessor.enabled=True \
  --inplace notebook.ipynb

# Or use nbstripout (installs as git filter)
pip install nbstripout
nbstripout --install
```

## Rules
- Restart kernel and run all cells before committing.
- Clear all outputs before git commit.
- Tag parameter cells for papermill.
- Each notebook should be self-contained and reproducible.
- Include `random_state` parameter for reproducibility.
- Use relative paths for data files.
