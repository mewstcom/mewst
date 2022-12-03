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
  match "/api/internal/follow/toggle", via: :post, as: :internal_api_follow_toggle,  to: "api/internal/follow/toggle/create#call"
  match "/api/internal/following",     via: :post, as: :internal_api_following_list, to: "api/internal/following/index#call"
  match "/home",                       via: :get,  as: :home,                        to: "home/show#call"
  match "/posts",                      via: :post, as: :post_list,                   to: "posts/create#call"
  match "/sign_in",                    via: :get,  as: :sign_in,                     to: "sign_in/new#call"
  match "/sign_in",                    via: :post,                                   to: "sign_in/create#call"
  match "/sign_in/confirmations",      via: :post, as: :sign_in_confirmation_list,   to: "sign_in/confirmations/create#call"
  match "/sign_in/confirmations/new",  via: :get,  as: :sign_in_new_confirmation,    to: "sign_in/confirmations/new#call"
  match "/sign_out",                   via: :get,  as: :sign_out,                    to: "sign_out/show#call"
  match "/sign_up",                    via: :get,  as: :sign_up,                     to: "sign_up/new#call"
  match "/sign_up",                    via: :post,                                   to: "sign_up/create#call"
  match "/sign_up/confirmations",      via: :post, as: :sign_up_confirmation_list,   to: "sign_up/confirmations/create#call"
  match "/sign_up/confirmations/new",  via: :get,  as: :sign_up_new_confirmation,    to: "sign_up/confirmations/new#call"
  match "/users",                      via: :post, as: :user_list,                   to: "users/create#call"
  match "/users/new",                  via: :get,  as: :new_user,                    to: "users/new#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
