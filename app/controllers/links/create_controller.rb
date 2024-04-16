# typed: true
# frozen_string_literal: true

class Links::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = LinkDataFetcherForm.new(form_params)

    if @form.invalid?
      return render("links/new/call", status: :unprocessable_entity)
    end

    result = LinkDataFetcher.new(target_url: @form.target_url).call

    if result.link
      @link = result.link
      return render
    end

    fetched_data = result.fetched_data
    @link_form = LinkForm.new(
      canonical_url: fetched_data&.canonical_url.presence || "",
      domain: fetched_data&.domain.presence || "",
      title: fetched_data&.title.presence || "",
      image_url: fetched_data&.image_url.presence || ""
    )

    if @link_form.invalid?
      @form.add_fetch_error!
      return render("links/new/call", status: :unprocessable_entity)
    end

    result = CreateLinkUseCase.new.call(
      canonical_url: @link_form.canonical_url,
      domain: @link_form.domain,
      title: @link_form.title,
      image_url: @link_form.image_url
    )

    @link = result.link
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:link_data_fetcher_form), ActionController::Parameters).permit(:target_url)
  end
end
