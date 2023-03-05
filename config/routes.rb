# typed: strict
# frozen_string_literal: true

require "sidekiq/web"

ROUTING_USERNAME_FORMAT = T.let(/[A-Za-z0-9_]+/, Regexp)

Rails.application.routes.draw do
  if Rails.env.development?
    mount Sidekiq::Web => "/sidekiq"
  end

  mount ImageUploader.derivation_endpoint => "/image"

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/@:atname",                                         via: :get,    as: :profile,                                          to: "profiles/show#call",                        atname: ROUTING_USERNAME_FORMAT
  match "/@:atname/posts/:post_id",                          via: :delete, as: :post,                                             to: "posts/destroy#call",                        atname: ROUTING_USERNAME_FORMAT
  match "/@:atname/posts/:post_id",                          via: :get,                                                           to: "posts/show#call",                           atname: ROUTING_USERNAME_FORMAT
  match "/accounts",                                         via: :post,   as: :account_list,                                     to: "accounts/create#call"
  match "/accounts/new",                                     via: :get,    as: :new_account,                                      to: "accounts/new#call"
  match "/api/internal/follow",                              via: :post,   as: :internal_api_follow,                              to: "api/internal/follow/create#call"
  match "/api/internal/following",                           via: :post,   as: :internal_api_following_list,                      to: "api/internal/following/index#call"
  match "/api/internal/posts",                               via: :post,   as: :internal_api_post_list,                           to: "api/internal/posts/create#call"
  match "/api/internal/pubsub/fanout_post/messages",         via: :post,   as: :internal_api_pubsub_fanout_post_message_list,     to: "api/internal/pubsub/fanout_post/messages/create#call"
  match "/api/internal/pubsub/add_post_to_home_timeline",    via: :post,   as: :internal_api_pubsub_add_post_to_home_timeline,    to: "api/internal/pubsub/add_post_to_home_timeline/create#call"
  match "/api/internal/unfollow",                            via: :post,   as: :internal_api_unfollow,                            to: "api/internal/unfollow/create#call"
  match "/twitter_friends",                                  via: :get,    as: :twitter_friend_list,                              to: "twitter_friends/index#call"
  match "/home",                                             via: :get,    as: :home,                                             to: "home/show#call"
  match "/settings",                                         via: :get,    as: :settings,                                         to: "settings/profiles/show#call"
  match "/settings",                                         via: :patch,                                                         to: "settings/profiles/update#call"
  match "/settings/account",                                 via: :get,    as: :settings_account,                                 to: "settings/accounts/show#call"
  match "/settings/account",                                 via: :patch,                                                         to: "settings/accounts/update#call"
  match "/settings/connected_accounts",                      via: :get,    as: :settings_connected_account_list,                  to: "settings/connected_accounts/index#call"
  match "/settings/twitter_account",                         via: :delete, as: :settings_twitter_account,                         to: "settings/twitter_accounts/destroy#call"
  match "/settings/twitter_authorization",                   via: :post,   as: :settings_twitter_authorization,                   to: "settings/twitter_authorizations/create#call"
  match "/settings/twitter_authorization/callback",          via: :get,    as: :settings_twitter_authorization_callback,          to: "settings/twitter_authorizations/callbacks/show#call"
  match "/sign_in",                                          via: :get,    as: :sign_in,                                          to: "sign_in/new#call"
  match "/sign_in",                                          via: :post,                                                          to: "sign_in/create#call"
  match "/sign_in/verification/phone_number/challenges",     via: :post,   as: :sign_in_verification_phone_number_challenge_list, to: "sign_in/verification/phone_number/challenges/create#call"
  match "/sign_in/verification/phone_number/challenges/new", via: :get,    as: :sign_in_verification_phone_number_new_challenge,  to: "sign_in/verification/phone_number/challenges/new#call"
  match "/sign_out",                                         via: :get,    as: :sign_out,                                         to: "sign_out/show#call"
  match "/sign_up",                                          via: :get,    as: :sign_up,                                          to: "sign_up/new#call"
  match "/sign_up",                                          via: :post,                                                          to: "sign_up/create#call"
  match "/sign_up/verification/phone_number/challenges",     via: :post,   as: :sign_up_verification_phone_number_challenge_list, to: "sign_up/verification/phone_number/challenges/create#call"
  match "/sign_up/verification/phone_number/challenges/new", via: :get,    as: :sign_up_verification_phone_number_new_challenge,  to: "sign_up/verification/phone_number/challenges/new#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
