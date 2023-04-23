# typed: strict
# frozen_string_literal: true

ROUTING_USERNAME_FORMAT = T.let(/[A-Za-z0-9_]+/, Regexp)

Rails.application.routes.draw do
  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/@:atname",                                         via: :get,    as: :profile,                                          to: "profiles/show#call",                        atname: ROUTING_USERNAME_FORMAT
  match "/@:atname/posts/:post_id",                          via: :delete, as: :post,                                             to: "posts/destroy#call",                        atname: ROUTING_USERNAME_FORMAT
  match "/@:atname/posts/:post_id",                          via: :get,                                                           to: "posts/show#call",                           atname: ROUTING_USERNAME_FORMAT
  match "/accounts",                                         via: :post,   as: :account_list,                                     to: "accounts/create#call"
  match "/accounts/new",                                     via: :get,    as: :new_account,                                      to: "accounts/new#call"
  match "/api/internal/follow",                              via: :post,   as: :internal_api_follow,                              to: "api/internal/follow/create#call"
  match "/api/internal/following",                           via: :post,   as: :internal_api_following_list,                      to: "api/internal/following/index#call"
  match "/api/internal/commented_posts",                     via: :post,   as: :internal_api_commented_post_list,                 to: "api/internal/commented_posts/create#call"
  match "/api/internal/pubsub/add_post_to_home_timeline",    via: :post,   as: :internal_api_pubsub_add_post_to_home_timeline,    to: "api/internal/pubsub/add_post_to_home_timeline#call"
  match "/api/internal/pubsub/fanout_post",                  via: :post,   as: :internal_api_pubsub_fanout_post,                  to: "api/internal/pubsub/fanout_post#call"
  match "/api/internal/tasks/send_verification_mail",        via: :post,   as: :internal_api_tasks_send_verification_mail,        to: "api/internal/tasks/send_verification_mail#call"
  match "/api/internal/unfollow",                            via: :post,   as: :internal_api_unfollow,                            to: "api/internal/unfollow/create#call"
  match "/fragment/posts/:post_id/repost_dropdown",          via: :get,    as: :fragment_repost_dropdown,                         to: "fragment/repost_dropdowns/show#call"
  match "/fragment/posts/:post_id/repost_dropdown_menu",     via: :get,    as: :fragment_repost_dropdown_menu,                    to: "fragment/repost_dropdown_menus/show#call"
  match "/home",                                             via: :get,    as: :home,                                             to: "home/show#call"
  match "/password",                                         via: :patch,  as: :password,                                         to: "passwords/update#call"
  match "/password/edit",                                    via: :get,    as: :edit_password,                                    to: "passwords/edit#call"
  match "/password_reset",                                   via: :get,    as: :password_reset,                                   to: "password_resets/new#call"
  match "/password_reset",                                   via: :post,                                                          to: "password_resets/create#call"
  match "/posts/:post_id/reposts",                           via: :post,   as: :repost_list,                                      to: "reposts/create#call"
  match "/settings",                                         via: :get,    as: :settings,                                         to: "settings/index#call"
  match "/settings/account",                                 via: :get,    as: :settings_account,                                 to: "settings/accounts/show#call"
  match "/settings/account",                                 via: :patch,                                                         to: "settings/accounts/update#call"
  match "/settings/profile",                                 via: :get,    as: :settings_profile,                                 to: "settings/profiles/show#call"
  match "/settings/profile",                                 via: :patch,                                                         to: "settings/profiles/update#call"
  match "/sign_in",                                          via: :get,    as: :sign_in,                                          to: "sign_in/new#call"
  match "/sign_in",                                          via: :post,                                                          to: "sign_in/create#call"
  match "/sign_in/verification/phone_number/challenges",     via: :post,   as: :sign_in_verification_phone_number_challenge_list, to: "sign_in/verification/phone_number/challenges/create#call"
  match "/sign_in/verification/phone_number/challenges/new", via: :get,    as: :sign_in_verification_phone_number_new_challenge,  to: "sign_in/verification/phone_number/challenges/new#call"
  match "/sign_out",                                         via: :get,    as: :sign_out,                                         to: "sign_out/show#call"
  match "/sign_up",                                          via: :get,    as: :sign_up,                                          to: "sign_up/new#call"
  match "/sign_up",                                          via: :post,                                                          to: "sign_up/create#call"
  match "/verification_challenges",                          via: :post,   as: :verification_challenge_list,                      to: "verification_challenges/create#call"
  match "/verification_challenges/new",                      via: :get,    as: :new_verification_challenge,                       to: "verification_challenges/new#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
