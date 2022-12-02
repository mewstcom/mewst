# typed: strict
# frozen_string_literal: true

require "sidekiq/web"

ROUTING_USERNAME_FORMAT = T.let(/[A-Za-z0-9_]+/, Regexp)

Rails.application.routes.draw do
  if Rails.env.development?
    mount Sidekiq::Web => "/sidekiq"
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/@:idname",                   via: :get,  as: :profile,                     to: "profiles/show#call",          idname: ROUTING_USERNAME_FORMAT
  match "/accounts",                   via: :post, as: :accounts,                    to: "accounts/create#call"
  match "/accounts/new",               via: :get,  as: :new_account,                 to: "accounts/new#call"
  match "/api/internal/follow/toggle", via: :post, as: :internal_api_follow_toggle,  to: "api/internal/follow/toggle/create#call"
  match "/api/internal/following",     via: :post, as: :internal_api_following_list, to: "api/internal/following/index#call"
  match "/home",                       via: :get,  as: :home,                        to: "home/show#call"
  match "/posts",                      via: :post, as: :post_list,                   to: "posts/create#call"
  match "/sign_in",                    via: :get,  as: :sign_in,                     to: "sign_in/new#call"
  match "/sign_in",                    via: :post,                                   to: "sign_in/create#call"
  match "/sign_in/callback",           via: :get,  as: :sign_in_callback,            to: "sign_in_callbacks/show#call"
  match "/sign_out",                   via: :get,  as: :sign_out,                    to: "sign_out/show#call"
  match "/sign_up",                    via: :get,  as: :sign_up,                     to: "sign_up/new#call"
  match "/sign_up",                    via: :post,                                   to: "sign_up/create#call"
  match "/sign_up/callback",           via: :get,  as: :sign_up_callback,            to: "sign_up_callbacks/show#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
