var Input = ReactBootstrap.Input;
var Table = ReactBootstrap.Table;
var ListGroup = ReactBootstrap.ListGroup;
var ListGroupItem = ReactBootstrap.ListGroupItem;
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
      rows.push(<ProxyRow key={proxies[i].name} container={this.props.container} rule={proxies[i]} reload={this.props.reload} />);
    }
    rows.push(<ProxyRow key="_" container={this.props.container} reload={this.props.reload} />);
    return (
      <Panel collapsible defaultExpanded header={this.props.container.name}>
        <Table fill striped bordered condensed hover>
          <thead>
            <tr>
              <th width="25%">Address</th>
              <th width="30%">Upstream Toxics</th>
              <th width="30%">Downstream Toxics</th>
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
    var state = {
      // modified: false,
      // adding: false,
      // updating: false,
      // removing: false,
    };
    var rule = this.props.rule || {};
    state.address = rule.upstream;
    var upstream = rule.upstream_toxics || {};
    state.upstream = {};
    state.upstream.latency = $.extend({}, upstream.latency);
    state.upstream.bandwidth = $.extend({}, upstream.bandwidth);
    var downstream = rule.downstream_toxics || {};
    state.downstream = {};
    state.downstream.latency = $.extend({}, downstream.latency);
    state.downstream.bandwidth = $.extend({}, downstream.bandwidth);
    return state;
  },
  propertyUpdateHandler: function(property) {
    var self = this;
    return function(event) {
      var newVal = event.target.type === "checkbox" ? event.target.checked : event.target.value;
      var state = $.extend(self.state, {modified: true});
      var parts = property.split(".");
      var currProp = state;
      for (var i=0; i < parts.length - 1; i++) {
        if (!currProp[parts[i]]) {
          currProp[parts[i]] = {};
        }
        currProp = currProp[parts[i]];
      }
      currProp[parts[parts.length - 1]] = newVal;
      self.setState(state);
    };
  },
  handleAdd: function() {
    var self = this;
    this.setState({adding: true});
    addProxy(this.props.container.name, this.state.address, function(proxy) {
      addToxic(proxy.name, "latency", true, {enabled: self.state.upstream.latency.enabled, latency: parseInt(self.state.upstream.latency.latency), jitter: 5});
      addToxic(proxy.name, "latency", false, {enabled: self.state.downstream.latency.enabled, latency: parseInt(self.state.downstream.latency.latency), jitter: 5});
      addToxic(proxy.name, "bandwidth", true, {enabled: self.state.upstream.bandwidth.enabled, rate: parseInt(self.state.upstream.bandwidth.rate)});
      addToxic(proxy.name, "bandwidth", false, {enabled: self.state.downstream.bandwidth.enabled, rate: parseInt(self.state.downstream.bandwidth.rate)});
      self.replaceState(self.getInitialState());
      self.props.reload();
    });
  },
  handleUpdate: function() {
    var self = this;
    this.setState({updating: true});
    // Just a remove/add...
    deleteProxy(this.props.rule.name, function() {
      addProxy(self.props.container.name, self.state.address, function(proxy) {
        addToxic(proxy.name, "latency", true, {enabled: self.state.upstream.latency.enabled, latency: parseInt(self.state.upstream.latency.latency), jitter: 5});
        addToxic(proxy.name, "latency", false, {enabled: self.state.downstream.latency.enabled, latency: parseInt(self.state.downstream.latency.latency), jitter: 5});
        addToxic(proxy.name, "bandwidth", true, {enabled: self.state.upstream.bandwidth.enabled, rate: parseInt(self.state.upstream.bandwidth.rate)});
        addToxic(proxy.name, "bandwidth", false, {enabled: self.state.downstream.bandwidth.enabled, rate: parseInt(self.state.downstream.bandwidth.rate)});
        self.setState({updating: false});
        self.props.reload();
      });
    });
  },
  handleRemove: function() {
    var self = this;
    this.setState({removing: true});
    deleteProxy(this.props.rule.name, function() {
      self.setState({removing: false});
      self.props.reload();
    });
  },
  render: function() {
    var submitting = this.state.updating || this.state.removing || this.state.adding;
    var buttons = this.props.rule ? [
      <Button key="update"
        bsStyle="warning"
        disabled={submitting}
        onClick={!submitting ? this.handleUpdate : null}>
        {!this.state.updating ? "Update" : "Updating..."}
      </Button>,
      <Button key="remove"
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
        <td><Input type="text" value={this.state.address} onChange={this.propertyUpdateHandler("address")} /></td>
        <td><ListGroup fill>
          <ListGroupItem>
            <Input type="text" label="Latency" value={this.state.upstream.latency.latency} onChange={this.propertyUpdateHandler("upstream.latency.latency")}
              addonBefore={<input type="checkbox" checked={this.state.upstream.latency.enabled} onChange={this.propertyUpdateHandler("upstream.latency.enabled")} />}
            />
          </ListGroupItem>
          <ListGroupItem>
            <Input type="text" label="Bandwidth" value={this.state.upstream.bandwidth.rate} onChange={this.propertyUpdateHandler("upstream.bandwidth.rate")}
              addonBefore={<input type="checkbox" checked={this.state.upstream.bandwidth.enabled} onChange={this.propertyUpdateHandler("upstream.bandwidth.enabled")} />}
            />
          </ListGroupItem>
        </ListGroup></td>
        <td><ListGroup fill>
          <ListGroupItem>
            <Input type="text" label="Latency" value={this.state.downstream.latency.latency} onChange={this.propertyUpdateHandler("downstream.latency.latency")}
              addonBefore={<input type="checkbox" checked={this.state.downstream.latency.enabled} onChange={this.propertyUpdateHandler("downstream.latency.enabled")} />}
            />
          </ListGroupItem>
          <ListGroupItem>
            <Input type="text" label="Bandwidth" value={this.state.downstream.bandwidth.rate} onChange={this.propertyUpdateHandler("downstream.bandwidth.rate")}
              addonBefore={<input type="checkbox" checked={this.state.downstream.bandwidth.enabled} onChange={this.propertyUpdateHandler("downstream.bandwidth.enabled")} />}
            />
          </ListGroupItem>
        </ListGroup></td>
        <td>
          {buttons}
        </td>
      </tr>
    );
  }
});

var controls = <ToxicControls />;

function addProxy(containerName, upstream, callback) {
  if (!upstream) {
    callback();
    return;
  }
  var upstreamParts = upstream.split(":");
  var ip = upstreamParts[0];
  var port = parseInt(upstreamParts[1]) || 80;
  $.ajax({
    url: "/api/proxies",
    method: "POST",
    contentType: "application/json",
    dataType: "json",
    data: JSON.stringify({container: containerName, ipAddress: ip, port: port}),
    success: function(data) {
      if (callback) {
        callback(data);
      }
    },
    error: function() {
      if (callback) {
        callback();
      }
    }
  });
}

function deleteProxy(name, callback) {
  $.ajax({
    url: "/api/proxies",
    method: "DELETE",
    contentType: "application/json",
    data: JSON.stringify({name: name}),
    complete: function() {
      if (callback) {
        callback();
      }
    }
  });
}

function addToxic(proxyName, toxicName, isUpstream, data, callback) {
  $.ajax({
    url: "/api/proxies/" + proxyName + "/toxics",
    method: "POST",
    contentType: "application/json",
    data: JSON.stringify({toxicName: toxicName, upstream: isUpstream, toxic: data}),
    complete: function() {
      if (callback) {
        callback();
      }
    }
  });
}

React.render(
  controls,
  document.getElementById("content")
);
