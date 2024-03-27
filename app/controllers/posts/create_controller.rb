# typed: true
# frozen_string_literal: true

class Posts::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new(form_params)

    if @form.invalid?
      return render("posts/new/call", status: :unprocessable_entity)
    end

    result = CreatePostUseCase.new.call(viewer: current_actor!, content: @form.content.not_nil!)

    unless @form.with_frame
      flash[:notice] = t("messages.posts.created")
      return redirect_to(home_path)
    end

    @post = result.post
    @stamp_checker = StampChecker.new(profile: current_actor!.profile, posts: [@post])
    @form = PostForm.new(with_frame: true)

    render(content_type: "text/vnd.turbo-stream.html", layout: false)
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:post_form), ActionController::Parameters).permit(:content, :with_frame)
  end
end
