# typed: true
# frozen_string_literal: true

class Links::NewController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = LinkDataFetcherForm.new(target_url: params[:url])
    @host_and_path = Url.new(params[:url]).shorten_host_and_path
  end
end
