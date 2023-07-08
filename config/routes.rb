# typed: strict
# frozen_string_literal: true

ROUTING_USERNAME_FORMAT = T.let(/[A-Za-z0-9_]+/, Regexp)

Rails.application.routes.draw do
  if Rails.env.development?
    mount GraphiQL::Rails::Engine, at: "/graphiql", graphql_path: "/graphql"
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/api/internal/pubsub/add_post_to_home_timeline",   via: :post,  as: :internal_api_pubsub_add_post_to_home_timeline,   to: "api/internal/pubsub/add_post_to_home_timeline#call"
  match "/api/internal/pubsub/fanout_post",                 via: :post,  as: :internal_api_pubsub_fanout_post,                 to: "api/internal/pubsub/fanout_post#call"
  match "/api/internal/tasks/send_email_confirmation_mail", via: :post,  as: :internal_api_tasks_send_email_confirmation_mail, to: "api/internal/tasks/send_email_confirmation_mail#call"
  match "/graphql/internal",                                via: :post,  as: :graphql_internal,                                to: "graphql/internal#call"
  match "/graphql/trunk",                                   via: :post,  as: :graphql_trunk,                                   to: "graphql/trunk#call"
  match "/latest/@:atname/timeline",                        via: :get,   as: :latest_timeline,                                 to: "latest/timeline/show#call", atname: ROUTING_USERNAME_FORMAT
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute
end
