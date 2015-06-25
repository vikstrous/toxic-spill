var ToxicControls = React.createClass({
  getInitialState: function() {
    return {containers: [
      {
        name: "backstabbing_sinoussi",
        proxyRules: [{
          address: "0.0.0.0",
          port: 80
        }]
      },
      {
        name: "gloomy_pasteur",
        proxyRules: []
      },
      {
        name: "silly_ptolemy",
        proxyRules: []
      }
      ]};
  },
  render: function() {
    var containerControls = [];
    for (var i=0; i < this.state.containers.length; i++) {
      var c = this.state.containers[i];
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
    return (
      <div>
        <div>{this.props.container.name}</div>
        <table border="1">
          <tr>
            <th>Address</th>
            <th>Port</th>
          </tr>
          {rows}
          <tr>

          </tr>
        </table>
        <br/>
        <br/>
      </div>
    );
  }
});

var ProxyRow = React.createClass({
  render: function() {
    return (
      <tr>
        <td>{this.props.rule.address}</td>
        <td>{this.props.rule.port}</td>
        <td><ReactBootstrap.Button bsStyle="danger">Remove</ReactBootstrap.Button></td>
      </tr>
    );
  }
});

React.render(
  <ToxicControls />,
  document.getElementById("content")
);
