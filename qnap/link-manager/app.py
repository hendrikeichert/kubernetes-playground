from flask import Flask, request, jsonify, render_template_string
import json
import os

app = Flask(__name__)

# Path to the JSON file for storing links
LINKS_FILE = 'data/links.json' # '/app/data/links.json'

# Initialize links.json if it doesn't exist
if not os.path.exists(LINKS_FILE):
    with open(LINKS_FILE, 'w') as f:
        json.dump([], f)

# Load links from JSON file
def load_links():
    with open(LINKS_FILE, 'r') as f:
        return json.load(f)

# Save links to JSON file
def save_links(links):
    with open(LINKS_FILE, 'w') as f:
        json.dump(links, f, indent=2)

# HTML template for the index page
INDEX_HTML = """
<!DOCTYPE html>
<html>
<head>
    <title>Link Manager - home.lan</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { text-align: center; }
        .link-list { margin-bottom: 20px; }
        .link-item { margin: 10px 0; }
        .form-container { border: 1px solid #ccc; padding: 20px; }
        input, button { margin: 5px; padding: 5px; }
        button { cursor: pointer; }
    </style>
</head>
<body>
    <h1>Link Manager</h1>
    <div class="link-list">
        <h2>Links</h2>
        {% if links %}
            <ul>
                {% for link in links %}
                    <li class="link-item">
                        <a href="{{ link.url }}" target="_blank">{{ link.name }}</a>
                        <form action="/delete/{{ loop.index0 }}" method="POST" style="display:inline;">
                            <button type="submit">Delete</button>
                        </form>
                    </li>
                {% endfor %}
            </ul>
        {% else %}
            <p>No links added yet.</p>
        {% endif %}
    </div>
    <div class="form-container">
        <h2>Add/Update Link</h2>
        <form action="/add" method="POST">
            <input type="text" name="name" placeholder="Link Name" required>
            <input type="url" name="url" placeholder="Link URL" required>
            <button type="submit">Add Link</button>
        </form>
    </div>
</body>
</html>
"""

@app.route('/')
def index():
    links = load_links()
    return render_template_string(INDEX_HTML, links=links)

@app.route('/add', methods=['POST'])
def add_link():
    links = load_links()
    name = request.form['name']
    url = request.form['url']
    # Check if link name exists, update if it does
    for link in links:
        if link['name'] == name:
            link['url'] = url
            save_links(links)
            return jsonify({'status': 'updated', 'name': name, 'url': url})
    # Add new link
    links.append({'name': name, 'url': url})
    save_links(links)
    return jsonify({'status': 'added', 'name': name, 'url': url})

@app.route('/delete/<int:index>', methods=['POST'])
def delete_link(index):
    links = load_links()
    if 0 <= index < len(links):
        removed = links.pop(index)
        save_links(links)
        return jsonify({'status': 'deleted', 'name': removed['name']})
    return jsonify({'status': 'error', 'message': 'Invalid index'})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=9080)
