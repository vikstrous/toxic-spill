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
        self.setState({containers: data});
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
    var proxies = this.props.container.proxies || [];
    for (var i=0; i < proxies.length; i++) {
      rows.push(<ProxyRow rule={proxies[i]}/>);
    }
    rows.push(<AddProxyRow />);
    return (
      <Panel collapsible defaultExpanded header={this.props.container.name}>
        <Table striped bordered condensed hover>
          <thead>
            <tr>
              <th>Listener</th>
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
      upstream: this.props.rule.upstream,
    };
  },
  handleUpstreamChange: function(event) {
    this.setState({
      modified: true,
      upstream: event.target.value
    });
  },
  handleUpdate: function() {
    this.setState({updating: true});
    // Not yet defined
    updateProxyRule(this.props.rule.name, this.state.upstream, function() {
      this.setState({updating: false});
    });
  },
  handleRemove: function() {
    this.setState({updating: true});
    // Not yet defined
    deleteProxyRule(this.props.rule.name);
  },
  render: function() {
    var submitting = this.state.updating || this.state.removing;
    return (
      <tr>
        <td><Input type="text" value={this.state.upstream} onChange={this.handleUpstreamChange} /></td>
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
        <td><Button bsStyle="success">Add</Button></td>
      </tr>
    );
  }
});

React.render(
  <ToxicControls />,
  document.getElementById("content")
);
