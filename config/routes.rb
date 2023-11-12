# typed: strict
# frozen_string_literal: true

ROUTING_USERNAME_FORMAT = T.let(/[A-Za-z0-9_]+/, Regexp)

Rails.application.routes.draw do
  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/internal/@:atname/posts",                                       via: :get,    as: :internal_profile_post_list,                      to: "internal/profiles/posts/index#call",                  atname: ROUTING_USERNAME_FORMAT
  match "/internal/accounts",                                             via: :post,   as: :internal_account_list,                           to: "internal/accounts/create#call"
  match "/internal/email_confirmations",                                  via: :post,   as: :internal_email_confirmation_list,                to: "internal/email_confirmations/create#call"
  match "/internal/email_confirmations/:email_confirmation_id",           via: :get,    as: :internal_email_confirmation,                     to: "internal/email_confirmations/show#call"
  match "/internal/email_confirmations/:email_confirmation_id/challenge", via: :post,   as: :internal_email_confirmation_challenge,           to: "internal/email_confirmations/challenges/create#call"
  match "/internal/posts/:post_id",                                       via: :get,    as: :internal_post,                                   to: "internal/posts/show#call"
  match "/internal/sessions",                                             via: :post,   as: :internal_session_list,                           to: "internal/sessions/create#call"
  match "/latest/@:atname/follow",                                        via: :delete, as: :latest_follow,                                   to: "latest/follows/destroy#call",                         atname: ROUTING_USERNAME_FORMAT
  match "/latest/@:atname/follow",                                        via: :post,                                                         to: "latest/follows/create#call",                          atname: ROUTING_USERNAME_FORMAT
  match "/latest/@:atname/posts",                                         via: :get,    as: :latest_profile_post_list,                        to: "latest/profiles/posts/index#call",                    atname: ROUTING_USERNAME_FORMAT
  match "/latest/notifications",                                          via: :get,    as: :latest_notification_list,                        to: "latest/notifications/index#call"
  match "/latest/posts",                                                  via: :post,   as: :latest_post_list,                                to: "latest/posts/create#call"
  match "/latest/posts/:post_id",                                         via: :delete, as: :latest_post,                                     to: "latest/posts/destroy#call"
  match "/latest/posts/:post_id",                                         via: :get,                                                          to: "latest/posts/show#call"
  match "/latest/posts/:post_id/stamp",                                   via: :delete, as: :latest_post_stamp,                               to: "latest/stamps/destroy#call"
  match "/latest/posts/:post_id/stamp",                                   via: :post,                                                         to: "latest/stamps/create#call"
  match "/latest/profiles/me",                                            via: :get,    as: :latest_profile_me,                               to: "latest/profiles/me/show#call"
  match "/latest/profiles/me",                                            via: :patch,                                                        to: "latest/profiles/me/update#call"
  match "/latest/suggested_profiles",                                     via: :get,    as: :suggested_profile_list,                          to: "latest/suggested_profiles/index#call"
  match "/latest/timeline",                                               via: :get,    as: :latest_timeline,                                 to: "latest/timeline/show#call"
  match "/latest/users/me",                                               via: :get,    as: :latest_user_me,                                  to: "latest/users/me/show#call"
  match "/latest/users/me",                                               via: :patch,                                                        to: "latest/users/me/update#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute
end
