# typed: strict
# frozen_string_literal: true

Rails.application.routes.draw do
  if Rails.env.development?
    mount GraphiQL::Rails::Engine, at: "/graphiql", graphql_path: "/graphql"
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/api/internal/pubsub/add_post_to_home_timeline", via: :post,  as: :internal_api_pubsub_add_post_to_home_timeline, to: "api/internal/pubsub/add_post_to_home_timeline#call"
  match "/api/internal/pubsub/fanout_post",               via: :post,  as: :internal_api_pubsub_fanout_post,               to: "api/internal/pubsub/fanout_post#call"
  match "/api/internal/tasks/send_verification_mail",     via: :post,  as: :internal_api_tasks_send_verification_mail,     to: "api/internal/tasks/send_verification_mail#call"
  match "/graphql/internal",                              via: :post,  as: :graphql_internal,                              to: "graphql/internal#call"
  match "/graphql/trunk",                                 via: :post,  as: :graphql_trunk,                                 to: "graphql/trunk#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute
end
