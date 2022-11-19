# typed: strict
# frozen_string_literal: true

require "sidekiq/web"

Rails.application.routes.draw do
  if Rails.env.development?
    mount Sidekiq::Web => "/sidekiq"
  end

  # match "/sign_in",                                             via: :get,    as: :new_user_session,                           to: "sign_in#new" # for Devise
  # match "/sign_in",                                             via: :get,    as: :sign_in,                                    to: "sign_in#new"

  match "/accounts",         via: :post, as: :accounts,         to: "accounts/create#call"
  match "/accounts/new",     via: :get,  as: :new_account,      to: "accounts/new#call"
  match "/home",             via: :get,  as: :home,             to: "home/show#call"
  match "/sign_out",         via: :get,  as: :sign_out,         to: "sign_out/show#call"
  match "/sign_up",          via: :get,  as: :sign_up,          to: "sign_up/new#call"
  match "/sign_up",          via: :post,                        to: "sign_up/create#call"
  match "/sign_in/callback", via: :get,  as: :sign_in_callback, to: "sign_in_callbacks/show#call"
  match "/sign_up/callback", via: :get,  as: :sign_up_callback, to: "sign_up_callbacks/show#call"

  root "welcome/show#call"
end
