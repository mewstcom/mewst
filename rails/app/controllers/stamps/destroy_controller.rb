# typed: true
# frozen_string_literal: true

class Stamps::DestroyController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = StampForm.new(target_post_id: params[:post_id])

    if @form.invalid?
      return render(
        "stamps/create/call",
        content_type: "text/vnd.turbo-stream.html",
        layout: false,
        status: :unprocessable_entity
      )
    end

    result = DeleteStampUseCase.new.call(viewer: viewer!, target_post: @form.target_post.not_nil!)
    @post = result.post
    @stamp_checker = StampChecker.new(profile: viewer!.profile_record, posts: [@post])

    render("stamps/create/call", content_type: "text/vnd.turbo-stream.html", layout: false)
  end
end
