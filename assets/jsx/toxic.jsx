var Input = ReactBootstrap.Input;
var Table = ReactBootstrap.Table;
var Panel = ReactBootstrap.Panel;
var Button = ReactBootstrap.Button;

var ToxicControls = React.createClass({
  reloadData: function() {
    var self = this;
    $.ajax({
      url: "/api/proxies",
      dataType: "json",
      success: function(data) {
        self.setState({containers: data});
      },
      error: function(xhr, status, err) {
        window.console.error(status, err.toString());
      }
    });
  },
  getInitialState: function() {
    return {};
  },
  componentDidMount: function() {
    this.reloadData();
  },
  render: function() {
    var containers = this.state.containers || [];
    var containerControls = [];
    for (var i=0; i < containers.length; i++) {
      var c = containers[i];
      containerControls.push(<ContainerControl key={i} container={c} reload={this.reloadData}/>);
    }
    return (
      <div>
        {containerControls}
      </div>
    );
  }
});

var ContainerControl = React.createClass({
  render: function() {
    var rows = [];
    var proxies = this.props.container.proxies || [];
    for (var i=0; i < proxies.length; i++) {
      rows.push(<ProxyRow container={this.props.container} rule={proxies[i]} reload={this.props.reload} />);
    }
    rows.push(<ProxyRow container={this.props.container} reload={this.props.reload} />);
    return (
      <Panel collapsible defaultExpanded header={this.props.container.name}>
        <Table striped bordered condensed hover>
          <thead>
            <tr>
              <th width="25%">Listener</th>
              <th width="30%">Upstream</th>
              <th width="30%">Downstream</th>
              <th width="15%">Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows}
          </tbody>
        </Table>
      </Panel>
    );
  }
});

var ProxyRow = React.createClass({
  getInitialState: function() {
    return {
      modified: false,
      adding: false,
      updating: false,
      removing: false,
      upstream: this.props.rule ? this.props.rule.upstream : "",
    };
  },
  handleUpstreamChange: function(event) {
    this.setState({
      modified: true,
      upstream: event.target.value
    });
  },
  handleAdd: function() {
    var self = this;
    this.setState({adding: true});
    addProxyRule(this.props.container.name, this.state.upstream, function() {
      self.setState({adding: false});
      self.props.reload();
    });
  },
  handleUpdate: function() {
    var self = this;
    this.setState({updating: true});
    // Not yet defined
    updateProxyRule(this.props.rule.name, this.state.upstream, function() {
      self.setState({updating: false});
      self.props.reload();
    });
  },
  handleRemove: function() {
    var self = this;
    this.setState({removing: true});
    deleteProxyRule(this.props.rule.name, function() {
      self.setState({removing: false});
      self.props.reload();
    });
  },
  render: function() {
    var submitting = this.state.updating || this.state.removing || this.state.adding;
    var buttons = this.props.rule ? [
      <Button
        bsStyle="warning"
        disabled={submitting}
        onClick={!submitting ? this.handleUpdate : null}>
        {!this.state.updating ? "Update" : "Updating..."}
      </Button>,
      <Button
        bsStyle="danger"
        disabled={submitting}
        onClick={!submitting ? this.handleRemove : null}>
        {!this.state.removing ? "Remove" : "Removing..."}
      </Button>
      ] :
      <Button
        bsStyle="success"
        disabled={this.state.adding}
        onClick={!this.state.adding ? this.handleAdd :null}>
          {!this.state.adding ? "Add" : "Adding..."}
      </Button>;
    return (
      <tr>
        <td><Input type="text" value={this.state.upstream} onChange={this.handleUpstreamChange} /></td>
        <td></td>
        <td></td>
        <td>
          {buttons}
        </td>
      </tr>
    );
  }
});

var controls = <ToxicControls />;

function addProxyRule(containerName, upstream, callback) {
  var upstreamParts = upstream.split(":");
  var ip = upstreamParts[0];
  var port = upstreamParts[1];
  $.ajax({
    url: "/api/proxies",
    method: "POST",
    contentType: "application/json",
    data: JSON.stringify({container: containerName, ipAddress: ip, port: parseInt(port)}),
    complete: function() {
      callback();
    }
  });
}

function deleteProxyRule(name, callback) {
  $.ajax({
    url: "/api/proxies",
    method: "DELETE",
    contentType: "application/json",
    data: JSON.stringify({name: name}),
    complete: function() {
      callback();
    }
  });
}

React.render(
  controls,
  document.getElementById("content")
);
