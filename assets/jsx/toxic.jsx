var Input = ReactBootstrap.Input;
var Table = ReactBootstrap.Table;
var Panel = ReactBootstrap.Panel;
var Button = ReactBootstrap.Button;

var ToxicControls = React.createClass({
  getInitialState: function() {
    return {};
  },
  componentDidMount: function() {
    var self = this;
    $.ajax({
      url: "/api/proxies",
      dataType: "json",
      success: function(data) {
        self.setState(data);
      },
      error: function(xhr, status, err) {
        window.console.error(status, err.toString());
      }
    });
  },
  render: function() {
    var containers = this.state.containers || [];
    var containerControls = [];
    for (var i=0; i < containers.length; i++) {
      var c = containers[i];
      containerControls.push(<ContainerControl key={i} container={c}/>);
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
    for (var i=0; i < this.props.container.proxyRules.length; i++) {
      rows.push(<ProxyRow rule={this.props.container.proxyRules[i]}/>);
    }
    rows.push(<AddProxyRow />);
    return (
      <Panel collapsible defaultExpanded header={this.props.container.name}>
        <Table striped bordered condensed hover>
          <thead>
            <tr>
              <th>Address</th>
              <th>Port</th>
              <th>Actions</th>
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
      updating: false,
      removing: false,
      address: this.props.rule.address,
      port: this.props.rule.port,
    };
  },
  handleAddressChange: function(event) {
    this.setState({
      modified: true,
      address: event.target.value
    });
  },
  handlePortChange: function(event) {
    this.setState({
      modified: true,
      port: event.target.value
    });
  },
  handleUpdate: function() {
    this.setState({updating: true});
    updateProxyRule(this.props.rule.id, this.state.address, this.state.port, function() {
      this.setState({updating: false});
    });
  },
  handleRemove: function() {
    this.setState({updating: true});
    deleteProxyRule(this.props.rule.id);
  },
  render: function() {
    var submitting = this.state.updating || this.state.removing;
    return (
      <tr>
        <td><Input type="text" value={this.state.address} onChange={this.handleAddressChange} /></td>
        <td><Input type="text" value={this.state.port} onChange={this.handlePortChange} /></td>
        <td>
          <Button
            bsStyle="warning"
            disabled={submitting}
            onClick={!submitting ? this.handleUpdate : null}>
            {!this.state.updating ? "Update" : "Updating..."}
          </Button>
          <Button
            bsStyle="danger"
            disabled={submitting}
            onClick={!submitting ? this.handleRemove : null}>
            {!this.state.removing ? "Remove" : "Removing..."}
          </Button>
        </td>
      </tr>
    );
  }
});

var AddProxyRow = React.createClass({
  render: function() {
    return (
      <tr>
        <td><Input type="text" /></td>
        <td><Input type="text" /></td>
        <td><Button bsStyle="success">Add</Button></td>
      </tr>
    );
  }
});

React.render(
  <ToxicControls />,
  document.getElementById("content")
);
