# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/@:atname/timeline", type: :request, api_version: :latest do
  context "when success" do
    let!(:user) { create(:user, :with_profile, :with_mewst_web_access_token) }
    let!(:profile) { user.first_profile }
    let!(:oauth_access_token) { user.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      form = Forms::CommentedPost.new(comment: "Hello", profile:)
      Services::CreateCommentedPost.new(form:).call
    end

    it "returns posts on timeline" do
      get("/latest/@#{profile.atname}/timeline", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expect(response.header["Link"]).to eq("")
    end
  end
end
